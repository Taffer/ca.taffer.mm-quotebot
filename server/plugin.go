package main

import (
	"fmt"
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

	active   bool      // Is the plugin currently active?
	lastPost time.Time // When did we last post a random quotation?
	channel  string    // The Channel ID of the channel we randomly post to.

	commandPattern *regexp.Regexp
}

// QuoteText - We track the quote and who said it.
type QuoteText struct {
	quote  string // Interesting (?) quotation.
	author string // Who's responsible for it?
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

	// ^/quote\s*(?P<command>(add|list|delete)\s*)?(?P<tail>.*)\s*$
	commandRegex string = `(?i)^` + slashTrigger + `\s*(?P<command>(add|list|delete)\s*)?(?P<tail>.*)\s*$`
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
	p.commandPattern = regexp.MustCompile(commandRegex)

	err := p.API.RegisterCommand(&model.Command{
		Trigger:          trigger,
		Description:      "Keep track of quotes and post them!",
		DisplayName:      pluginName,
		AutoComplete:     true,
		AutoCompleteDesc: "📜 Keep track of quotes and post them! Use `/" + trigger + " help` for usage.",
		AutoCompleteHint: "help",
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
			// If tail is a number, show that quote.
			num, err := strconv.Atoi(tail)
			if err == nil {
				response, responseError = p.ShowQuote(num)
			} else {
				response, responseError = p.ShowHelp()
			}
		}
	} else {
		switch strings.ToLower(strings.TrimSpace(command)) {
		case "add":
			// Anyone can add quotes.
			if len(tail) > 0 {
				response, responseError = p.AddQuote(tail)
			} else {
				response = nil
				responseError = p.NewError("Empty quote.", "Try adding a quote with some text.", "QuotebotPlugin.ExecuteCommand")
			}

		case "help":
			// Anyone can ask for help.
			response, responseError = p.ShowHelp()

		case "list":
			// List all known quotes. Admins only.
			response, responseError = p.ListQuotes()

		case "delete":
			// Delete a quote specified by tail as a number. Admins only.
			num, err := strconv.Atoi(tail)
			if err == nil {
				response, responseError = p.DeleteQuote(num)
			} else {
				response, responseError = p.ShowHelp()
			}
		}
	}

	return response, responseError
}

// MessageHasBeenPosted - can use this to periodically post a quote if people are talking...
func (p *QuotebotPlugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if p.active == false { // Is this even possible?
		return
	}

	// TODO: Ignore messages from bots.
	if post.ChannelId == p.channel {
		p.PostRandom()
	}
}

// UserHasJoinedChannel - another trigger for periodically posting
func (p *QuotebotPlugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	if p.active == false { // Is this even possible?
		return
	}

	if channelMember.ChannelId == p.channel {
		p.PostRandom()
	}
}

// UserHasLeftChannel - another trigger for periodically posting
func (p *QuotebotPlugin) UserHasLeftChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	if p.active == false { // Is this even possible?
		return
	}

	if channelMember.ChannelId == p.channel {
		p.PostRandom()
	}
}

// -----------------------------------------------------------------------------
// Quotebot functions
// -----------------------------------------------------------------------------

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
		Where:         where,
	}
}

// PostRandom - Post a random quotation if enough time has passed.
func (p *QuotebotPlugin) PostRandom() {
	now := time.Now()
	delta := now.Sub(p.lastPost)

	if delta.Minutes() > p.configuration.postDelta {
		p.lastPost = now

		// TODO: post a random quote to p.configuration.postChannel.
		// p.channel is the Channel ID of p.configuration.postChannel.
	}
}

// ShowQuote - Post the specified quote.
func (p *QuotebotPlugin) ShowQuote(num int) (*model.CommandResponse, *model.AppError) {
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, fmt.Sprintf("`ShowQuote(%v)`", num)), nil
}

// ShowRandom - Show a random quotation in response to a command.
func (p *QuotebotPlugin) ShowRandom() (*model.CommandResponse, *model.AppError) {
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "`ShowRandom()`"), nil
}

// ShowHelp - Post the usage instructions.
func (p *QuotebotPlugin) ShowHelp() (*model.CommandResponse, *model.AppError) {
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "`ShowHelp()`"), nil
}

// AddQuote - Add the given quote to the quote database.
func (p *QuotebotPlugin) AddQuote(quote string) (*model.CommandResponse, *model.AppError) {
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, fmt.Sprintf("`AddQuote(%q)`", quote)), nil
}

// ListQuotes - List the known quotes.
func (p *QuotebotPlugin) ListQuotes() (*model.CommandResponse, *model.AppError) {
	// TODO: How do I know if the user is an admin in this team?
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "`ListQuotes()`"), nil
}

// DeleteQuote - Delete the specified quote.
func (p *QuotebotPlugin) DeleteQuote(num int) (*model.CommandResponse, *model.AppError) {
	// TODO: How do I know if the user is an admin in this team?
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, fmt.Sprintf("`DeleteQuote(%v)`", num)), nil
}
