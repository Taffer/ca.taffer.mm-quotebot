package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
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
	userID    string    // User ID of the user we randomly post as (p.configuration.postUser).
	channelID string    // The Channel ID of the channel we randomly post to (p.configuration.postChannel).
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

	// ^/quote\s*(?P<command>(add|channel|delete|info|interval|list)\s*)?(?P<tail>.*)\s*$
	// TODO: Remove "debug" when we're done with it.
	commandRegex string = `(?i)^` + slashTrigger + `\s*(?P<command>(debug|add|channel|delete|info|interval|list)\s*)?(?P<tail>.*)\s*$`

	// I still haven't looked into i18n.
	helpText = `Quotebot remembers quotes you tell it about, and spits them out again when you ask it to.

Commands:

* /quote - Regurgitate a random quote.
* /quote *x* - Show quote number *x*.
* /quote add *genius quote* - Store *genius quote* for later. Don't forget to
  include an attribution!
* /quote help - Show the help.
* /quote info - Show the number of quotes, the channel, and the interval.`
	adminHelpText = `Admin commands:

* /quote channel *x* - Monitor channel *x* for activity and randomly
  show quotes there.
* /quote delete *x* - Delete quote number *x*.
* /quote interval *x* - The time between automatically posting quotes
  in a channel.
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

// PostRandom - Post a random quotation if enough time has passed.
func (p *QuotebotPlugin) PostRandom() {
	now := time.Now()
	delta := now.Sub(p.lastPost)

	if delta.Minutes() < p.configuration.postDelta {
		return
	}

	p.lastPost = now

	var quote string
	switch len(p.quotes) {
	case 0:
		// something zen
		quote = "There is no void if you don't try to fill it. -- Marty Rubin"
	case 1:
		// rand.Intn() throws an exception if you call it with 0.
		quote = p.quotes[0]
	default:
		quote = p.quotes[rand.Intn(len(p.quotes))+1]
	}

	// Example of using CreatPost() from the unit tests:
	//
	// https://github.com/mattermost/mattermost-server/blob/master/api4/post_test.go#L31
	//
	// Also in the demo plugin:
	//
	// https://github.com/mattermost/mattermost-plugin-demo/blob/master/server/channel_hooks.go#L20
	//
	// Requres UserID, ChannelID, message.

	post, err := p.API.CreatePost(&model.Post{
		UserId:    p.userID,
		ChannelId: p.channelID,
		Message:   quote,
	})
	if post == nil {
		p.API.LogError("PostRandom() - post came back nil.")
	}
	if err != nil {
		p.API.LogError("PostRandom() - error: %q", err)
	}
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

// SaveQuotes - Load the quote list from the key-value store.
func (p *QuotebotPlugin) SaveQuotes() *model.AppError {
	raw, err := json.Marshal(p.quotes)
	if err != nil {
		return p.NewError("Unable to save quotes.", fmt.Sprintf("json.Marshal(%q) failed.", p.quotes), "SaveQuotes")
	}

	return p.API.KVSet("quotes", raw)
}
