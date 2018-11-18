package main

import (
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
	commandRegex string = `(?i)^` + slashTrigger + `\s*(?P<command>(debug|add|delete|list)\s*)?(?P<tail>.*)\s*$`

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
	p.configuration = new(configuration)
	p.commandPattern = regexp.MustCompile(commandRegex)

	err := p.API.RegisterCommand(&model.Command{
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

		// case "channel": // Admins only.
		// 	// Tell the bot which channel to monitor.
		// 	if len(tail) > 0 {
		// 		// Attempt to find "tail" as a channel name.
		// 		response, responseError = p.SetChannel(args.UserId, tail, args.TeamId)
		// 	} else {
		// 		response = nil
		// 		responseError = p.NewError("Empty channel.", "You have to specify a channel name.", "ExecuteCommand")
		// 	}

		case "delete": // Admins only.
			// Delete a quote specified by tail as a number.
			response, responseError = p.DeleteQuote(args.UserId, tail)

		case "help":
			// Anyone can ask for help.
			response, responseError = p.ShowHelp()

		// case "interval": // Admins only.
		// 	// Change the posting interval, in minutes.
		// 	num, err := strconv.Atoi(tail)
		// 	if err == nil {
		// 		response, responseError = p.SetInterval(args.UserId, num)
		// 	} else {
		// 		response = nil
		// 		responseError = p.NewError("Invalid interval.", "You have to specify an interval, in minutes.",
		// 			"ExecuteCommand")
		// 	}

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

// -----------------------------------------------------------------------------
// Quotebot commands
// -----------------------------------------------------------------------------

// AddQuote - Add the given quote to the quote database.
func (p *QuotebotPlugin) AddQuote(quote string) (*model.CommandResponse, *model.AppError) {
	if len(quote) < 1 {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Empty quote. Try adding a quote with some text."), nil
	}

	// TODO: Should we search the list for "quote" before adding it?
	p.configuration.quotes = append(p.configuration.quotes, quote)

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL,
		fmt.Sprintf("Added %q as quote number %d.", quote, len(p.configuration.quotes))), nil
}

// DeleteQuote - Delete the specified quote.
func (p *QuotebotPlugin) DeleteQuote(userID string, tail string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) {
		num, err := strconv.Atoi(tail)
		if err != nil {
			return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "What quote? You have to specify a quote index."), nil
		}

		quoteIdx := num - 1 // The list is 1-based for humans.
		if quoteIdx < 0 || len(p.configuration.quotes) == 0 || quoteIdx >= len(p.configuration.quotes) {
			return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
				fmt.Sprintf("You can't delete quote %d, it doesn't exist.", num)), nil
		}

		p.configuration.quotes = append(p.configuration.quotes[:quoteIdx], p.configuration.quotes[quoteIdx+1:]...)

		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("Deleted quote %d. There are %d quotes on file.", num, len(p.configuration.quotes))), nil
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Only admins can delete quotes."), nil
}

// ListQuotes - List the known quotes.
func (p *QuotebotPlugin) ListQuotes(userID string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) {
		response := fmt.Sprintf("There are %d quotes on file.", len(p.configuration.quotes))

		for idx := range p.configuration.quotes {
			// The list is 1-based for humans.
			response += fmt.Sprintf("\n* %d = %q", idx+1, p.configuration.quotes[idx])
		}

		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, response), nil
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Only admins can list the quotes."), nil
}

// // SetChannel - Set the channel the bot monitors.
// func (p *QuotebotPlugin) SetChannel(userID string, channel string, teamID string) (*model.CommandResponse, *model.AppError) {
// 	if p.IsAdmin(userID) {
// 		// This seems to always fail?
// 		newChannel, err := p.API.GetChannelByName(channel, teamID, false) // What if the channel doesn't exist?
// 		if err != nil {
// 			return nil, p.NewError("Channel doesn't exist.", fmt.Sprintf("%q isn't a valid channel, use one that exists.", channel),
// 				"SetChannel")
// 		}

// 		p.configuration.postChannel = newChannel.Id
// 		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Channel set to %s.", newChannel.DisplayName)), nil
// 	}

// 	return nil, p.NewError("Can't set channel.", "Only admins can set the channel.", "SetChannel")
// }

// // SetInterval - Set the response interval, in minutes.
// func (p *QuotebotPlugin) SetInterval(userID string, interval int) (*model.CommandResponse, *model.AppError) {
// 	if p.IsAdmin(userID) {
// 		if interval < 15 {
// 			return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
// 				"You can't set an Interval less than 15 minutes, it's annoying."), nil
// 		}

// 		p.configuration.postDelta = float64(interval)
// 		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
// 			fmt.Sprintf("Interval set to %v minutes.", p.configuration.postDelta)), nil
// 	}

// 	return nil, p.NewError("Can't set interval.", "Only admins can set the interval.", "SetInterval")
// }

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

	if len(p.configuration.quotes) == 0 {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "There aren't any quotes yet."), nil
	} else if len(p.configuration.quotes) < num {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("Unable to show quote %v, it doesn't exist yet. There are %d quotes on file.", num,
				len(p.configuration.quotes))), nil
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, fmt.Sprintf("> %v", p.configuration.quotes[num-1])), nil
}

// ShowRandom - Show a random quotation in response to a command.
func (p *QuotebotPlugin) ShowRandom() (*model.CommandResponse, *model.AppError) {
	numQuotes := len(p.configuration.quotes)
	if numQuotes == 1 {
		return p.ShowQuote("1")
	}
	if numQuotes > 1 {
		// rand.Intn() throws an exception if you call it with 0...
		return p.ShowQuote(fmt.Sprintf("%d", rand.Intn(numQuotes)+1))
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "There aren't any quotes yet."), nil
}
