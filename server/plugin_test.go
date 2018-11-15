package main

import (
	"regexp"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// Test support utilities
// -----------------------------------------------------------------------------

func runTestPluginCommand(t *testing.T, cmd string) (*model.CommandResponse, *model.AppError) {
	p := initTestPlugin(t)
	assert.Nil(t, p.OnActivate())

	var command *model.CommandArgs
	command = &model.CommandArgs{
		Command: cmd,
	}

	return p.ExecuteCommand(&plugin.Context{}, command)
}

func initTestPlugin(t *testing.T) *QuotebotPlugin {
	api := &plugintest.API{}
	api.On("RegisterCommand", mock.Anything).Return(nil)
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil)
	api.On("GetUser", mock.Anything).Return(&model.User{
		Id:        "userid",
		Nickname:  "User",
		Username:  "hunter2",
		FirstName: "User",
		LastName:  "McUserface",
	}, (*model.AppError)(nil))

	p := QuotebotPlugin{}
	p.SetAPI(api)

	return &p
}

// -----------------------------------------------------------------------------
// Tests - Utility functions.
// -----------------------------------------------------------------------------

// TestFindNamedSubstrings - Make sure our regex and FindNamedSubstrings work.
func TestFindNamedSubstrings(t *testing.T) {
	pat := regexp.MustCompile(commandRegex)

	// Regular commands.
	matches := FindNamedSubstrings(pat, "/quote")
	command := matches["command"]
	assert.EqualValues(t, command, "")
	tail := matches["tail"]
	assert.EqualValues(t, tail, "")

	matches = FindNamedSubstrings(pat, "/quote 1")
	command = matches["command"]
	assert.EqualValues(t, command, "")
	tail = matches["tail"]
	assert.EqualValues(t, tail, "1")

	matches = FindNamedSubstrings(pat, "/quote add")
	command = matches["command"]
	assert.EqualValues(t, command, "add")
	tail = matches["tail"]
	assert.EqualValues(t, tail, "")

	matches = FindNamedSubstrings(pat, "/quote add Some genius quote.")
	command = matches["command"]
	assert.EqualValues(t, command, "add ")
	tail = matches["tail"]
	assert.EqualValues(t, tail, "Some genius quote.")

	// Admin commands.
	matches = FindNamedSubstrings(pat, "/quote list")
	command = matches["command"]
	assert.EqualValues(t, command, "list")
	tail = matches["tail"]
	assert.EqualValues(t, tail, "")

	matches = FindNamedSubstrings(pat, "/quote delete")
	command = matches["command"]
	assert.EqualValues(t, command, "delete")
	tail = matches["tail"]
	assert.EqualValues(t, tail, "")

	matches = FindNamedSubstrings(pat, "/quote delete 1")
	command = matches["command"]
	assert.EqualValues(t, command, "delete ")
	tail = matches["tail"]
	assert.EqualValues(t, tail, "1")
}

// -----------------------------------------------------------------------------
// Tests - Plugin callbacks.
// -----------------------------------------------------------------------------

// TestOnActivate - Test the OnActivate callback.
func TestOnActivate(t *testing.T) {
	p := initTestPlugin(t)

	assert.False(t, p.active)
	p.OnActivate()
	assert.True(t, p.active)
}

// TestOnDeactivate - Test the OnDeactivate callback.
func TestOnDeactivate(t *testing.T) {
	p := initTestPlugin(t)

	assert.False(t, p.active)
	p.OnActivate()
	assert.True(t, p.active)
	p.OnDeactivate()
	assert.False(t, p.active)
}

// TestExecuteCommand - Test the ExecuteCommand callback.
func TestExecuteCommand(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/quote")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowRandom()`")

	resp, err = runTestPluginCommand(t, "/quote ")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowRandom()`")

	resp, err = runTestPluginCommand(t, "/quote help")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowHelp()`")

	resp, err = runTestPluginCommand(t, "/quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowQuote(1)`")

	resp, err = runTestPluginCommand(t, "/quote add some genius quote")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`AddQuote(\"some genius quote\")`")

	resp, err = runTestPluginCommand(t, "/quote list")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ListQuotes()`")

	resp, err = runTestPluginCommand(t, "/quote delete 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`DeleteQuote(1)`")

	resp, err = runTestPluginCommand(t, "/quote delete")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowHelp()`")
}

// // TestMessageHasBeenPosted - Test the MessageHasBeenPosted callback.
// func TestMessageHasBeenPosted(t *testing.T) {
// }

// // TestUserHasJoinedChannel - Test the UserHasJoinedChannel callback.
// func TestUserHasJoinedChannel(t *testing.T) {
// }

// // TestUserHasLeftChannel - Test the UserHasLeftChannel callback.
// func TestUserHasLeftChannel(t *testing.T) {
// }

// -----------------------------------------------------------------------------
// Tests - Quotebot functions
// -----------------------------------------------------------------------------

// TestNewResponse - Test the NewResponse function.
func TestNewResponse(t *testing.T) {
	p := initTestPlugin(t)

	resp := p.NewResponse("type", "string")
	assert.EqualValues(t, resp.ResponseType, "type")
	assert.EqualValues(t, resp.Text, "string")
}

// TestNewError - Test the NewError function.
func TestNewError(t *testing.T) {
	p := initTestPlugin(t)

	err := p.NewError("message", "details", "where")
	assert.EqualValues(t, err.Message, "message")
	assert.EqualValues(t, err.DetailedError, "details")
	assert.EqualValues(t, err.Where, "where")
}

// // TestPostRandom - Test the PostRandom function.
// func TestPostRandom(t *testing.T) {
// }

// // TestShowQuote - Test the ShowQuote function.
// func TestShowQuote(t *testing.T) {
// }

// // TestShowRandom - Test the ShowRandom function.
// func TestShowRandom(t *testing.T) {
// }

// // TestShowHelp - Test the ShowHelp function.
// func TestShowHelp(t *testing.T) {
// }

// // TestAddQuote - Test the AddQuote function.
// func TestAddQuote(t *testing.T) {
// }

// // TestListQuotes - Test the ListQuotes function.
// func TestListQuotes(t *testing.T) {
// }

// // TestDeleteQuote - Test the DeleteQuote function.
// func TestDeleteQuote(t *testing.T) {
// }
