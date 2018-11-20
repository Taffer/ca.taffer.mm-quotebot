package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// QuotebotPlugin for Mattermost
type QuotebotPlugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	active    bool      // Is the plugin currently active?
	lastPost  time.Time // When did we last post a random quotation?
	channelID string    // The Channel ID of the channel we randomly post to.
	quotes    []string  // The list of quotes we know about.

	commandPattern *regexp.Regexp
}

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

const (
	pluginPath string = "plugins/ca.taffer.mm-quotebot"
	iconFile   string = "if_chat_1055095.png"
	iconPath   string = pluginPath + "/" + iconFile
	iconURI    string = "/" + iconPath

	trigger      string = "quote"
	slashTrigger string = "/" + trigger
	pluginName   string = "Quotebot"

	// ^/quote\s*(?P<command>(add|channel|delete|interval|list)\s*)?(?P<tail>.*)\s*$
	// TODO: Remove "debug" when we're done with it.
	// TODO: Add "channel" and "interval" back later.
	commandRegex string = `(?i)^` + slashTrigger + `\s*(?P<command>(debug|add|channel|delete|interval|list)\s*)?(?P<tail>.*)\s*$`

	// I still haven't looked into i18n.
	// TODO: Add "channel" and "interval" (and the posting blurb) back later.
	helpText = `Quotebot remembers quotes you tell it about, and spits them out again when you ask it to.

Commands:

* /quote - Regurgitate a random quote.
* /quote *x* - Show quote number *x*.
* /quote add *genius quote* - Store *genius quote* for later. Don't forget to include an attribution!
* /quote help - Show the help.

Admin commands:

* /quote delete *x* - Delete quote number *x*.
* /quote list - List all known quotes.`
)

// -----------------------------------------------------------------------------
// Utility functions.
// -----------------------------------------------------------------------------

// FindNamedSubstrings - Return a map of named matches.
func FindNamedSubstrings(re *regexp.Regexp, candidate string) map[string]string {
	found := make(map[string]string)

	values := re.FindStringSubmatch(candidate)
	keys := re.SubexpNames()

	// Why do you start indexing keys at 1 instead of 0?
	for idx := 1; idx < len(keys); idx++ {
		found[keys[idx]] = values[idx]
	}

	return found
}

// -----------------------------------------------------------------------------
// Plugin callbacks
// -----------------------------------------------------------------------------

// OnActivate - Plugin has been activated.
func (p *QuotebotPlugin) OnActivate() error {
	p.active = true

	configuration := new(configuration)
	err := p.loadConfiguration(configuration)
	if err != nil {
		return err
	}
	p.setConfiguration(configuration)

	p.commandPattern = regexp.MustCompile(commandRegex)

	err = p.LoadQuotes() // Prime the quote cannon!

	err = p.API.RegisterCommand(&model.Command{
		Trigger:          trigger,
		Description:      "Keep track of quotes and post them!",
		DisplayName:      pluginName,
		AutoComplete:     true,
		AutoCompleteDesc: "📜 Keep track of quotes and post them! Use `/" + trigger + " help` for usage.",
		AutoCompleteHint: "[add quotation | #]",
		IconURL:          iconURI,
	})

	return err
}

// OnDeactivate - Plugin has been deactivated.
func (p *QuotebotPlugin) OnDeactivate() error {
	p.active = false

	return nil
}

// ExecuteCommand - Plugin needs to run a command, maybe.
func (p *QuotebotPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if p.active == false { // Is this even possible?
		return nil, nil
	}

	if p.commandPattern.MatchString(args.Command) == false {
		// It's not for us.
		return nil, nil
	}

	// Dig out the command and tail.
	matches := FindNamedSubstrings(p.commandPattern, args.Command)
	command := matches["command"]
	tail := matches["tail"]

	var response *model.CommandResponse
	var responseError *model.AppError

	if command == "" {
		if tail == "" {
			// "/quote" - Show a random quote.
			response, responseError = p.ShowRandom()
		} else {
			response, responseError = p.ShowQuote(tail)
		}
	} else {
		switch strings.ToLower(strings.TrimSpace(command)) {
		case "debug": // TODO: DELETE ME WHEN DONE.
			user, err := p.API.GetUser(args.UserId)
			if err == nil {
				response = p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
					fmt.Sprintf("User.Roles is: %q", user.Roles))
				responseError = nil
			} else {
				response = p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "GetUser() fail. I have no idea.")
				responseError = nil
			}

		case "add":
			// Anyone can add quotes.
			response, responseError = p.AddQuote(tail)

		case "channel": // Admins only.
			// Tell the bot which channel to monitor.
			response, responseError = p.SetChannel(args.UserId, tail, args.TeamId)

		case "delete": // Admins only.
			// Delete a quote specified by tail as a number.
			response, responseError = p.DeleteQuote(args.UserId, tail)

		case "help":
			// Anyone can ask for help.
			response, responseError = p.ShowHelp()

		case "interval": // Admins only.
			// Change the posting interval, in minutes.
			response, responseError = p.SetInterval(args.UserId, tail)

		case "list":
			// List all known quotes. Admins only.
			response, responseError = p.ListQuotes(args.UserId)
		}
	}

	return response, responseError
}

// // MessageHasBeenPosted - can use this to periodically post a quote if people are talking...
// func (p *QuotebotPlugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
// 	if p.active == false { // Is this even possible?
// 		return
// 	}

// 	// TODO: Ignore messages from bots.
// 	if post.ChannelId == p.channelID {
// 		p.PostRandom()
// 	}
// }

// // UserHasJoinedChannel - another trigger for periodically posting
// func (p *QuotebotPlugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
// 	if p.active == false { // Is this even possible?
// 		return
// 	}

// 	if channelMember.ChannelId == p.channelID {
// 		p.PostRandom()
// 	}
// }

// // UserHasLeftChannel - another trigger for periodically posting
// func (p *QuotebotPlugin) UserHasLeftChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
// 	if p.active == false { // Is this even possible?
// 		return
// 	}

// 	if channelMember.ChannelId == p.channelID {
// 		p.PostRandom()
// 	}
// }

// -----------------------------------------------------------------------------
// Quotebot functions
// -----------------------------------------------------------------------------

// IsAdmin - Is the given UserID an admin user?
func (p *QuotebotPlugin) IsAdmin(userID string) bool {
	isAdmin := false
	user, err := p.API.GetUser(userID)
	if err == nil {
		roles := strings.Fields(user.Roles)
		for idx := range roles {
			switch roles[idx] {
			case model.PERMISSIONS_CHANNEL_ADMIN:
				isAdmin = true
			case model.PERMISSIONS_TEAM_ADMIN:
				isAdmin = true
			case model.PERMISSIONS_SYSTEM_ADMIN:
				isAdmin = true
			}
		}
	}

	return isAdmin
}

// LoadQuotes - Load the quote list from the key-value store.
func (p *QuotebotPlugin) LoadQuotes() *model.AppError {
	raw, err := p.API.KVGet("quotes")
	if raw == nil {
		// Stay empty.
		return nil
	}
	if err != nil {
		// message string, details string, where string
		return p.NewError("Unable to load quotes.", "API.KVGet() failed.", "LoadQuotes")
	}

	var quotes []string
	loadErr := json.Unmarshal(raw, &quotes)
	if loadErr != nil {
		return p.NewError("Unable to load quotes.", fmt.Sprintf("json.Unmarshal(%q) failed.", raw), "LoadQuotes")
	}

	p.quotes = quotes

	return nil
}

// NewResponse - Create a new response object.
func (p *QuotebotPlugin) NewResponse(responseType string, responseText string) *model.CommandResponse {
	props := map[string]interface{}{
		"from_webhook":  "true",
		"use_user_icon": "true",
	}

	return &model.CommandResponse{
		ResponseType: responseType,
		Username:     pluginName,
		Text:         responseText,
		Props:        props,
		IconURL:      iconURI,
	}
}

// NewError - Create a new error object.
func (p *QuotebotPlugin) NewError(message string, details string, where string) *model.AppError {
	return &model.AppError{
		Message:       message,
		DetailedError: details,
		Where:         "QuotebotPlugin." + where,
	}
}

// // PostRandom - Post a random quotation if enough time has passed.
// func (p *QuotebotPlugin) PostRandom() {
// 	now := time.Now()
// 	delta := now.Sub(p.lastPost)

// 	if delta.Minutes() > p.configuration.postDelta {
// 		p.lastPost = now

// 		// TODO: post a random quote to p.configuration.postChannel.
// 		// p.channelID is the Channel ID of p.configuration.postChannel.
// 	}
// }

// SaveQuotes - Load the quote list from the key-value store.
func (p *QuotebotPlugin) SaveQuotes() *model.AppError {
	raw, err := json.Marshal(p.quotes)
	if err != nil {
		return p.NewError("Unable to save quotes.", fmt.Sprintf("json.Marshal(%q) failed.", p.quotes), "SaveQuotes")
	}

	return p.API.KVSet("quotes", raw)
}

// -----------------------------------------------------------------------------
// Quotebot commands
// -----------------------------------------------------------------------------

// AddQuote - Add the given quote to the quote database.
func (p *QuotebotPlugin) AddQuote(quote string) (*model.CommandResponse, *model.AppError) {
	if len(quote) < 1 {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Empty quote. Try adding a quote with some text."), nil
	}

	// TODO: Should we search the list for "quote" before adding it?
	p.quotes = append(p.quotes, quote)
	err := p.SaveQuotes()
	if err != nil {
		return nil, err
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		fmt.Sprintf("Added %q as quote number %d.", quote, len(p.quotes))), nil
}

// DeleteQuote - Delete the specified quote.
func (p *QuotebotPlugin) DeleteQuote(userID string, tail string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) == false {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Only admins can delete quotes."), nil
	}

	num, err := strconv.Atoi(tail)
	if err != nil {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "What quote? You have to specify a quote index."), nil
	}

	quoteIdx := num - 1 // The list is 1-based for humans.
	if quoteIdx < 0 || len(p.quotes) == 0 || quoteIdx >= len(p.quotes) {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("You can't delete quote %d, it doesn't exist.", num)), nil
	}

	p.quotes = append(p.quotes[:quoteIdx], p.quotes[quoteIdx+1:]...)
	saveErr := p.SaveQuotes()
	if saveErr != nil {
		return nil, saveErr
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		fmt.Sprintf("Deleted quote %d. There are %d quotes on file.", num, len(p.quotes))), nil
}

// ListQuotes - List the known quotes.
func (p *QuotebotPlugin) ListQuotes(userID string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) == false {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Only admins can list the quotes."), nil
	}

	response := fmt.Sprintf("There are %d quotes on file.", len(p.quotes))

	for idx := range p.quotes {
		// The list is 1-based for humans.
		response += fmt.Sprintf("\n* %d = %q", idx+1, p.quotes[idx])
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, response), nil
}

// SetChannel - Set the channel the bot monitors.
func (p *QuotebotPlugin) SetChannel(userID string, channel string, teamID string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) == false {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Only admins can set the channel."), nil
	}

	if len(channel) == 0 {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "You must specify a channel name."), nil
	}

	newChannel, err := p.API.GetChannelByName(channel, teamID, false) // What if the channel doesn't exist?
	if err != nil {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("%q isn't a valid channel, use one that exists.", channel)), nil
	}

	p.configuration.postChannel = newChannel.Id
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Channel set to %s.", newChannel.DisplayName)), nil
}

// SetInterval - Set the response interval, in minutes.
func (p *QuotebotPlugin) SetInterval(userID string, tail string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) == false {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Only admins can set the interval."), nil
	}

	interval, err := strconv.Atoi(tail)
	if err != nil {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			"You have to specify an interval in minutes, >= 15."), nil
	}

	if interval < 15 {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			"You can't set an Interval less than 15 minutes, it's annoying."), nil
	}

	p.configuration.postDelta = float64(interval)
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		fmt.Sprintf("Interval set to %v minutes.", p.configuration.postDelta)), nil
}

// ShowHelp - Post the usage instructions.
func (p *QuotebotPlugin) ShowHelp() (*model.CommandResponse, *model.AppError) {
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, helpText), nil
}

// ShowQuote - Post the specified quote.
func (p *QuotebotPlugin) ShowQuote(tail string) (*model.CommandResponse, *model.AppError) {
	// If tail is a number, show that quote.
	num, err := strconv.Atoi(tail)
	if err != nil {
		return p.ShowHelp()
	}

	if len(p.quotes) == 0 {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "There aren't any quotes yet."), nil
	} else if len(p.quotes) < num {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("Unable to show quote %v, it doesn't exist yet. There are %d quotes on file.", num,
				len(p.quotes))), nil
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, fmt.Sprintf("> %v", p.quotes[num-1])), nil
}

// ShowRandom - Show a random quotation in response to a command.
func (p *QuotebotPlugin) ShowRandom() (*model.CommandResponse, *model.AppError) {
	numQuotes := len(p.quotes)
	if numQuotes == 1 {
		return p.ShowQuote("1")
	}
	if numQuotes > 1 {
		// rand.Intn() throws an exception if you call it with 0...
		return p.ShowQuote(fmt.Sprintf("%d", rand.Intn(numQuotes)+1))
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "There aren't any quotes yet."), nil
}
