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
		TeamId:  "teamid",
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
		TeamId:  "teamid",
	}

	return p.ExecuteCommand(&plugin.Context{}, command)
}

func initAPI(t *testing.T, user string) *plugintest.API {
	api := &plugintest.API{}

	// Things that don't change.
	api.On("RegisterCommand", mock.Anything).Return(nil)
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil)
	api.On("LoadPluginConfiguration", mock.Anything).Return(nil)

	api.On("GetChannelByName", mock.Anything, mock.Anything, mock.Anything).Return(&model.Channel{
		Id:          "some ID string",
		DisplayName: "mock",
	}, (*model.AppError)(nil))

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

	return api
}

func initTestPlugin(t *testing.T, user string) *QuotebotPlugin {
	api := initAPI(t, user)
	api.On("KVGet", mock.Anything).Return(nil, nil)
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil)

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
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = runTestPluginCommand(t, "/quote ")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = runTestPluginCommand(t, "/quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = runTestPluginCommand(t, "/quote add")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Empty quote. Try adding a quote with some text.")

	resp, err = runTestPluginCommand(t, "/quote add some genius quote")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Added \"some genius quote\" as quote number 1.")

	resp, err = runTestPluginCommand(t, "/quote help")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, helpText)

	// Admin commands.
	resp, err = runTestPluginCommand(t, "/quote channel")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the channel.")

	resp, err = runTestPluginCommand(t, "/quote channel ~town-square")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the channel.")

	resp, err = runTestPluginCommand(t, "/quote delete")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can delete quotes.")

	resp, err = runTestPluginCommand(t, "/quote delete 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can delete quotes.")

	resp, err = runTestPluginCommand(t, "/quote interval")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the interval.")

	resp, err = runTestPluginCommand(t, "/quote interval 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the interval.")

	resp, err = runTestPluginCommand(t, "/quote list")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can list the quotes.")
}

// TestExecuteCommandAdmin - Test the ExecuteCommand() triggers that require admin access.
func TestExecuteCommandAdmin(t *testing.T) {
	resp, err := runTestPluginCommandAdmin(t, "/quote channel")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You must specify a channel name.")

	resp, err = runTestPluginCommandAdmin(t, "/quote channel ~town-square")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Channel set to mock.")

	resp, err = runTestPluginCommandAdmin(t, "/quote delete")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "What quote? You have to specify a quote index.")

	// TODO: Test this with a list of actual quotes.
	resp, err = runTestPluginCommandAdmin(t, "/quote delete -1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You can't delete quote -1, it doesn't exist.")

	resp, err = runTestPluginCommandAdmin(t, "/quote delete 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You can't delete quote 1, it doesn't exist.")

	resp, err = runTestPluginCommandAdmin(t, "/quote interval")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You have to specify an interval in minutes, >= 15.")

	resp, err = runTestPluginCommandAdmin(t, "/quote interval 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You can't set an Interval less than 15 minutes, it's annoying.")

	resp, err = runTestPluginCommandAdmin(t, "/quote interval 120")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Interval set to 120 minutes.")

	resp, err = runTestPluginCommandAdmin(t, "/quote list")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "There are 0 quotes on file.")
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

// TestLoadQuotes - Test the LoadQuotes function.
func TestLoadQuotes(t *testing.T) {
	p := initTestPlugin(t, "normal")
	err := p.LoadQuotes()
	assert.Nil(t, err)
	assert.EqualValues(t, len(p.quotes), 0)

	api := initAPI(t, "normal")
	api.On("KVGet", mock.Anything).Return([]byte(`["quote 1", "quote 2"]`), nil)
	p.SetAPI(api)
	err = p.LoadQuotes()
	assert.Nil(t, err)
	assert.EqualValues(t, len(p.quotes), 2)
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

// TestSaveQuotes - Test the SaveQuotes function.
// This test is weak. Not sure how to mock this to test it in a useful way,
// it's a fairly trivial function...
func TestSaveQuotes(t *testing.T) {
	p := initTestPlugin(t, "normal")
	assert.Nil(t, p.OnActivate())
	assert.EqualValues(t, len(p.quotes), 0)
}

// TestShowQuote - Test the ShowQuote function.
func TestShowQuote(t *testing.T) {
	p := initTestPlugin(t, "normal")
	assert.Nil(t, p.OnActivate())
	assert.EqualValues(t, len(p.quotes), 0)

	resp, err := p.ShowQuote("foo") // "" calls ShowRandom() instead of ShowQuote().
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, helpText)

	resp, err = p.ShowQuote("1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = p.AddQuote("quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)

	resp, err = p.ShowQuote("1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_IN_CHANNEL)
	assert.EqualValues(t, resp.Text, "> quote 1")

	resp, err = p.ShowQuote("2")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Unable to show quote 2, it doesn't exist yet. There are 1 quotes on file.")
}

// TestShowRandom - Test the ShowRandom function.
func TestShowRandom(t *testing.T) {
	p := initTestPlugin(t, "normal")
	assert.Nil(t, p.OnActivate())
	assert.EqualValues(t, len(p.quotes), 0)

	resp, err := p.ShowRandom()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = p.AddQuote("quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)

	resp, err = p.ShowRandom()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_IN_CHANNEL)
	assert.EqualValues(t, resp.Text, "> quote 1")
}

// TestShowHelp - Test the ShowHelp function.
func TestShowHelp(t *testing.T) {
	p := initTestPlugin(t, "normal")

	resp, err := p.ShowHelp()
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, helpText)
}

// TestAddQuote - Test the AddQuote function.
func TestAddQuote(t *testing.T) {
	p := initTestPlugin(t, "normal")
	assert.Nil(t, p.OnActivate())
	assert.EqualValues(t, len(p.quotes), 0)

	resp, err := p.AddQuote("")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Empty quote. Try adding a quote with some text.")
	assert.EqualValues(t, len(p.quotes), 0)

	resp, err = p.AddQuote("quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_IN_CHANNEL)
	assert.EqualValues(t, resp.Text, "Added \"quote 1\" as quote number 1.")
	assert.EqualValues(t, len(p.quotes), 1)
}

// TestListQuotes - Test the ListQuotes function.
func TestListQuotes(t *testing.T) {
	// Regular user testing.
	p := initTestPlugin(t, "normal")
	assert.Nil(t, p.OnActivate())

	resp, err := p.ListQuotes("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Only admins can list the quotes.")

	// Admin testing.
	p = initTestPlugin(t, "team")
	assert.Nil(t, p.OnActivate())

	resp, err = p.ListQuotes("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "There are 0 quotes on file.")

	resp, err = p.AddQuote("quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)

	resp, err = p.ListQuotes("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "There are 1 quotes on file.\n* 1 = \"quote 1\"")

	resp, err = p.AddQuote("quote 2")
	assert.NotNil(t, resp)
	assert.Nil(t, err)

	resp, err = p.ListQuotes("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "There are 2 quotes on file.\n* 1 = \"quote 1\"\n* 2 = \"quote 2\"")
}

// TestDeleteQuote - Test the DeleteQuote function.
func TestDeleteQuote(t *testing.T) {
	// Regular user testing.
	p := initTestPlugin(t, "normal")
	assert.Nil(t, p.OnActivate())

	resp, err := p.DeleteQuote("userid", "1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Only admins can delete quotes.")

	// Admin testing.
	p = initTestPlugin(t, "team")
	assert.Nil(t, p.OnActivate())

	resp, err = p.DeleteQuote("userid", "")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "What quote? You have to specify a quote index.")

	resp, err = p.DeleteQuote("userid", "1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You can't delete quote 1, it doesn't exist.")

	resp, err = p.AddQuote("quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)

	resp, err = p.DeleteQuote("userid", "1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Deleted quote 1. There are 0 quotes on file.")
}
