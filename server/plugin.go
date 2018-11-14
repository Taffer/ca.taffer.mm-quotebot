package main

import (
	"fmt"
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
)

// -----------------------------------------------------------------------------
// Plugin callbacks
// -----------------------------------------------------------------------------

// OnActivate - Plugin has been activated.
func (p *QuotebotPlugin) OnActivate() error {
	p.active = true

	return nil
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

	if strings.HasPrefix(args.Command, slashTrigger) == false {
		// It's not for us.
		return nil, nil
	}

	var response *model.CommandResponse
	var responseError *model.AppError

	if args.Command == slashTrigger {
		// "/quote" - Show a random quote.
		response, responseError = p.ShowRandom()
	} else {
		parts := strings.Fields(args.Command[len(slashTrigger):])

		switch strings.ToLower(parts[0]) {
		case "add":
			// Anyone can add quotes.
			response, responseError = p.AddQuote(strings.TrimSpace(args.Command[strings.Index(args.Command, parts[0])+len(parts[0]):]))

		case "help":
			// Anyone can ask for help.
			response, responseError = p.ShowHelp()

		case "list":
			// List all known quotes. Admins only.
			response, responseError = p.ListQuotes()

		case "delete":
			// Delete a quote specified by parts[1] as a number. Admins only.
			num, _ := strconv.Atoi(parts[1])
			response, responseError = p.DeleteQuote(num)

		default:
			// If parts[1] is a number, show that quote.
			num, err := strconv.Atoi(parts[1])
			if err == nil {
				response, responseError = p.ShowQuote(num)
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
	return *model.AppError{
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
