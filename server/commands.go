package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

// -----------------------------------------------------------------------------
// Quotebot admin-only commands
// -----------------------------------------------------------------------------

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

	if len(channel) == 0 || channel == "~" {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "You must specify a channel name."), nil
	}

	if channel[0:1] == "~" {
		channel = channel[1:]
	}

	newChannel, err := p.API.GetChannelByName(teamID, channel, false) // What if the channel doesn't exist?
	if err != nil {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			fmt.Sprintf("%q isn't a valid channel, use one that exists.", channel)), nil
	}

	p.configuration.postChannel = newChannel.DisplayName
	p.channelID = newChannel.Id
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
	if interval > 10080 { // It's been... one week...
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			"You can't set the Interval to more than a week, that's excessive."), nil
	}

	p.configuration.postDelta = float64(interval)
	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		fmt.Sprintf("Interval set to %v minutes.", p.configuration.postDelta)), nil
}

// SetUser - Set the (bot) account name used to post random quotes.
func (p *QuotebotPlugin) SetUser(userID string, userName string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) == false {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Only admins can set the post user."), nil
	}

	if len(userName) == 0 || userName == "@" {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "You must specify a user name."), nil
	}

	if userName[0:1] == "@" {
		userName = userName[1:]
	}

	// Make sure this user is a user before we go crazy.
	user, err := p.API.GetUserByUsername(userName)
	if err != nil {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Unable to find a user named %q.", userName)), nil
	}

	p.userID = user.Id
	p.configuration.postUser = user.Username

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Quotes will be posted by %v.", user.Username)), nil
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

// ShowHelp - Post the usage instructions.
func (p *QuotebotPlugin) ShowHelp(userID string) (*model.CommandResponse, *model.AppError) {
	if p.IsAdmin(userID) {
		return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, strings.Join([]string{helpText, adminHelpText}, "\n\n")), nil
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, helpText), nil
}

// ShowInfo - Show plug info.
// This function is an i18n nightmare, but at least it's short...
func (p *QuotebotPlugin) ShowInfo(userID string) (*model.CommandResponse, *model.AppError) {
	info := "You are a"
	if p.IsAdmin(userID) {
		info += "n Admin."
	} else {
		info += " User."
	}

	info += fmt.Sprintf(" Quotebot knows %v quotes.", len(p.quotes))

	channel, err := p.API.GetChannel(p.channelID)
	if err == nil {
		info += fmt.Sprintf(" Monitoring %s for activity every %v minutes.", channel.DisplayName, p.configuration.postDelta)
	} else {
		info += fmt.Sprintf(" Monitoring a non-existent channel. An Admin should fix that.")
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, info), nil
}

// ShowQuote - Post the specified quote.
func (p *QuotebotPlugin) ShowQuote(userID string, tail string) (*model.CommandResponse, *model.AppError) {
	// If tail is a number, show that quote.
	num, err := strconv.Atoi(tail)
	if err != nil {
		return p.ShowHelp(userID)
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
func (p *QuotebotPlugin) ShowRandom(userID string) (*model.CommandResponse, *model.AppError) {
	numQuotes := len(p.quotes)
	if numQuotes == 1 {
		return p.ShowQuote(userID, "1")
	}
	if numQuotes > 1 {
		// rand.Intn() throws an exception if you call it with 0...
		return p.ShowQuote(userID, fmt.Sprintf("%d", rand.Intn(numQuotes)+1))
	}

	return p.NewResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "There aren't any quotes yet."), nil
}
