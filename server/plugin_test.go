package main

import (
	"regexp"
	"strings"
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
	p := initTestPlugin(t, "normal")
	assert.Nil(t, p.OnActivate())

	var command *model.CommandArgs
	command = &model.CommandArgs{
		Command: cmd,
		UserId:  "userid",
	}

	return p.ExecuteCommand(&plugin.Context{}, command)
}

func runTestPluginCommandAdmin(t *testing.T, cmd string) (*model.CommandResponse, *model.AppError) {
	p := initTestPlugin(t, "system")
	assert.Nil(t, p.OnActivate())

	var command *model.CommandArgs
	command = &model.CommandArgs{
		Command: cmd,
		UserId:  "userid",
	}

	return p.ExecuteCommand(&plugin.Context{}, command)
}

func initTestPlugin(t *testing.T, user string) *QuotebotPlugin {
	api := &plugintest.API{}
	api.On("RegisterCommand", mock.Anything).Return(nil)
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil)

	switch user {
	case "channel": // Channel admin.
		api.On("GetUser", mock.Anything).Return(&model.User{
			Id:        "userid",
			Nickname:  "User",
			Username:  "hunter2",
			FirstName: "User",
			LastName:  "McUserface",
			Roles:     strings.Join([]string{"system_user", model.PERMISSIONS_CHANNEL_ADMIN}, " "),
		}, (*model.AppError)(nil))

	case "team": // Team admin.
		api.On("GetUser", mock.Anything).Return(&model.User{
			Id:        "userid",
			Nickname:  "User",
			Username:  "hunter2",
			FirstName: "User",
			LastName:  "McUserface",
			Roles:     strings.Join([]string{"system_user", model.PERMISSIONS_TEAM_ADMIN}, " "),
		}, (*model.AppError)(nil))

	case "system": // System admin.
		api.On("GetUser", mock.Anything).Return(&model.User{
			Id:        "userid",
			Nickname:  "User",
			Username:  "hunter2",
			FirstName: "User",
			LastName:  "McUserface",
			Roles:     strings.Join([]string{"system_user", model.PERMISSIONS_SYSTEM_ADMIN}, " "),
		}, (*model.AppError)(nil))

	case "everything": // Overkill!
		api.On("GetUser", mock.Anything).Return(&model.User{
			Id:        "userid",
			Nickname:  "User",
			Username:  "hunter2",
			FirstName: "User",
			LastName:  "McUserface",
			Roles: strings.Join([]string{"system_user", model.PERMISSIONS_CHANNEL_ADMIN, model.PERMISSIONS_TEAM_ADMIN,
				model.PERMISSIONS_SYSTEM_ADMIN}, " "),
		}, (*model.AppError)(nil))

	default:
		api.On("GetUser", mock.Anything).Return(&model.User{
			Id:        "userid",
			Nickname:  "User",
			Username:  "hunter2",
			FirstName: "User",
			LastName:  "McUserface",
			Roles:     "system_user",
		}, (*model.AppError)(nil))
	}

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
	p := initTestPlugin(t, "normal")

	assert.False(t, p.active)
	p.OnActivate()
	assert.True(t, p.active)
}

// TestOnDeactivate - Test the OnDeactivate callback.
func TestOnDeactivate(t *testing.T) {
	p := initTestPlugin(t, "normal")

	assert.False(t, p.active)
	p.OnActivate()
	assert.True(t, p.active)
	p.OnDeactivate()
	assert.False(t, p.active)
}

// TestExecuteCommand - Test the ExecuteCommand callback.
func TestExecuteCommand(t *testing.T) {
	// Normal user commands.
	resp, err := runTestPluginCommand(t, "/quote")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowRandom()`")

	resp, err = runTestPluginCommand(t, "/quote ")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowRandom()`")

	resp, err = runTestPluginCommand(t, "/quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ShowQuote(1)`")

	resp, err = runTestPluginCommand(t, "/quote add")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Empty quote.")

	resp, err = runTestPluginCommand(t, "/quote add some genius quote")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`AddQuote(\"some genius quote\")`")

	resp, err = runTestPluginCommand(t, "/quote help")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, helpText)

	// Admin commands.
	resp, err = runTestPluginCommand(t, "/quote channel")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Empty channel.")

	resp, err = runTestPluginCommand(t, "/quote channel ~town-square")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Can't set channel.")

	resp, err = runTestPluginCommand(t, "/quote delete")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "What quote?")

	resp, err = runTestPluginCommand(t, "/quote delete 1")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Can't delete.")

	resp, err = runTestPluginCommand(t, "/quote interval")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Invalid interval.")

	resp, err = runTestPluginCommand(t, "/quote interval 1")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Can't set interval.")

	resp, err = runTestPluginCommand(t, "/quote list")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Can't list.")
}

// TestExecuteCommandAdmin - Test the ExecuteCommand() triggers that require admin access.
func TestExecuteCommandAdmin(t *testing.T) {
	resp, err := runTestPluginCommandAdmin(t, "/quote channel")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Empty channel.")

	resp, err = runTestPluginCommandAdmin(t, "/quote channel ~town-square")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`SetChannel(~town-square)`")

	resp, err = runTestPluginCommandAdmin(t, "/quote delete")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "What quote?")

	resp, err = runTestPluginCommandAdmin(t, "/quote delete 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`DeleteQuote(1)`")

	resp, err = runTestPluginCommandAdmin(t, "/quote interval")
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Message, "Invalid interval.")

	resp, err = runTestPluginCommandAdmin(t, "/quote interval 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`SetInterval(1)`")

	resp, err = runTestPluginCommandAdmin(t, "/quote list")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "`ListQuotes()`")
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

// TestIsAdmin - Test whether I can tell if you're an admin or not.
func TestIsAdmin(t *testing.T) {
	p := initTestPlugin(t, "normal")
	assert.False(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "channel")
	assert.True(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "team")
	assert.True(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "system")
	assert.True(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "everything")
	assert.True(t, p.IsAdmin("userid"))
}

// TestNewResponse - Test the NewResponse function.
func TestNewResponse(t *testing.T) {
	p := initTestPlugin(t, "normal")

	resp := p.NewResponse("type", "string")
	assert.EqualValues(t, resp.ResponseType, "type")
	assert.EqualValues(t, resp.Text, "string")
}

// TestNewError - Test the NewError function.
func TestNewError(t *testing.T) {
	p := initTestPlugin(t, "normal")

	err := p.NewError("message", "details", "where")
	assert.EqualValues(t, err.Message, "message")
	assert.EqualValues(t, err.DetailedError, "details")
	assert.EqualValues(t, err.Where, "QuotebotPlugin.where")
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

// TestShowHelp - Test the ShowHelp function.
func TestShowHelp(t *testing.T) {
	p := initTestPlugin(t, "normal")

	resp, err := p.ShowHelp()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, helpText)
}

// // TestAddQuote - Test the AddQuote function.
// func TestAddQuote(t *testing.T) {
// }

// // TestListQuotes - Test the ListQuotes function.
// func TestListQuotes(t *testing.T) {
// }

// // TestDeleteQuote - Test the DeleteQuote function.
// func TestDeleteQuote(t *testing.T) {
// }
