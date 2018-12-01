package main

import (
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

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

	// Defaults.
	if configuration.postDelta == 0 {
		configuration.postDelta = 15
	}
	if configuration.postChannel == "" {
		configuration.postChannel = "town-square"
	}
	if configuration.postUser == "" {
		configuration.postUser = "Taffer" // This is unlikely to exist, you're set for failure.
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
			response, responseError = p.ShowRandom(args.UserId)
		} else {
			response, responseError = p.ShowQuote(args.UserId, tail)
		}
	} else {
		switch strings.ToLower(strings.TrimSpace(command)) {
		case "debug": // TODO: DELETE ME WHEN DONE.
			p.PostRandom()
			response = p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "PostRandom() is done.")
			responseError = nil

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
			response, responseError = p.ShowHelp(args.UserId)

		case "info":
			// Anyone can ask for the info.
			response, responseError = p.ShowInfo(args.UserId)

		case "interval": // Admins only.
			// Change the posting interval, in minutes.
			response, responseError = p.SetInterval(args.UserId, tail)

		case "list":
			// List all known quotes. Admins only.
			response, responseError = p.ListQuotes(args.UserId)

		case "user":
			// Set the bot account used for posting. Admins only.
			response, responseError = p.SetUser(args.UserId, tail)
		}
	}

	return response, responseError
}

// MessageHasBeenPosted - can use this to periodically post a quote if people are talking...
func (p *QuotebotPlugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if p.active == false { // Is this even possible?
		return
	}

	// TODO: How to ignore posts from bots?
	if post.ChannelId != p.channelID { // Ignore posts in other channels.
		return
	}

	// Post a random quote, maybe.
	p.PostRandom()
}

// UserHasJoinedChannel - another trigger for periodically posting
func (p *QuotebotPlugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	if p.active == false { // Is this even possible?
		return
	}

	if channelMember.ChannelId != p.channelID { // Ignore posts in other channels.
		return
	}

	// Post a random quote, maybe.
	p.PostRandom()
}

// UserHasLeftChannel - another trigger for periodically posting
func (p *QuotebotPlugin) UserHasLeftChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	if p.active == false { // Is this even possible?
		return
	}

	if channelMember.ChannelId != p.channelID { // Ignore posts in other channels.
		return
	}

	// Post a random quote, maybe.
	p.PostRandom()
}
