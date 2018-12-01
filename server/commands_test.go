package main

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

// One of the things I don't like about Go is how things are implied by
// conventions (that don't seem to be documented very well) rather than
// explicitly specified.
//
// For example, these private functions need to go in the alphabetically
// first *_test.go file or the other *_test.go files can't find them.
//
// Unless, of course, this is just how the Mattermost plugin sample project
// is set up.

// -----------------------------------------------------------------------------
// Test support utilities
// -----------------------------------------------------------------------------

func testUser(user string) *model.User {
	fakeUser := &model.User{
		Id:        "userid",
		Nickname:  user,
		Username:  "Someone",
		FirstName: user,
		LastName:  "User",
		Roles:     "",
	}

	switch user {
	case "channel": // Channel admin.
		fakeUser.Roles = strings.Join([]string{"system_user", model.PERMISSIONS_CHANNEL_ADMIN}, " ")

	case "team": // Team admin.
		fakeUser.Roles = strings.Join([]string{"system_user", model.PERMISSIONS_TEAM_ADMIN}, " ")

	case "system": // System admin.
		fakeUser.Roles = strings.Join([]string{"system_user", model.PERMISSIONS_SYSTEM_ADMIN}, " ")

	case "everything": // Overkill!
		fakeUser.Roles = strings.Join([]string{"system_user", model.PERMISSIONS_CHANNEL_ADMIN, model.PERMISSIONS_TEAM_ADMIN,
			model.PERMISSIONS_SYSTEM_ADMIN}, " ")

	default:
		fakeUser.Roles = "system_user"
	}

	return fakeUser
}

func testChannel(channelID string) (*model.Channel, *model.AppError) {
	var fakeChannel *model.Channel
	var fakeErr *model.AppError

	switch channelID {
	case "fail":
		fakeChannel = nil
		fakeErr = &model.AppError{
			Message:       "Nope.",
			DetailedError: "Nope.",
			Where:         "QuotebotPlugin.Nope.",
		}

	default:
		fakeChannel = &model.Channel{
			Id:          "some ID string",
			DisplayName: "mock",
		}
		fakeErr = nil
	}

	return fakeChannel, fakeErr
}

func runTestPluginCommand(t *testing.T, cmd string, user string, channelID string) (*model.CommandResponse, *model.AppError) {
	p := initTestPlugin(t, user, channelID)
	assert.Nil(t, p.OnActivate())

	var command *model.CommandArgs
	command = &model.CommandArgs{
		Command: cmd,
		UserId:  "userid",
		TeamId:  "teamid",
	}

	return p.ExecuteCommand(&plugin.Context{}, command)
}

func initAPI(t *testing.T, user string, channelID string, quotesRaw []byte) *plugintest.API {
	api := &plugintest.API{}
	fakeUser := testUser(user)
	fakeChannel, fakeChannelErr := testChannel(channelID)

	// Things that don't change depending on user/channelID.
	api.On("RegisterCommand", mock.Anything).Return(nil)
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil)
	api.On("LoadPluginConfiguration", mock.Anything).Return(nil)
	api.On("KVGet", mock.Anything).Return(quotesRaw, nil)
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil)

	// These need specific mocks.
	api.On("GetUser", mock.Anything).Return(fakeUser, (*model.AppError)(nil))
	api.On("GetChannelByName", mock.Anything, mock.Anything, mock.Anything).Return(fakeChannel, fakeChannelErr)
	api.On("GetChannel", mock.Anything).Return(fakeChannel, fakeChannelErr)

	return api
}

func initTestPlugin(t *testing.T, user string, channelID string) *QuotebotPlugin {
	api := initAPI(t, user, channelID, nil)

	p := QuotebotPlugin{}
	p.SetAPI(api)

	return &p
}

// -----------------------------------------------------------------------------
// Quotebot admin-only commands
// -----------------------------------------------------------------------------

// TestDeleteQuote - Test the DeleteQuote function.
func TestDeleteQuote(t *testing.T) {
	// Regular user testing.
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err := p.DeleteQuote("userid", "1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Only admins can delete quotes.")

	// Admin testing.
	p = initTestPlugin(t, "team", "mock")
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

// TestListQuotes - Test the ListQuotes function.
func TestListQuotes(t *testing.T) {
	// Regular user testing.
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err := p.ListQuotes("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Only admins can list the quotes.")

	// Admin testing.
	p = initTestPlugin(t, "team", "mock")
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

// TestSetChannel - test the SetChannel function.
func TestSetChannel(t *testing.T) {
	// Regular user testing.
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err := p.SetChannel("userid", "town-square", "teamid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Only admins can set the channel.")

	// Admin testing.
	p = initTestPlugin(t, "team", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err = p.SetChannel("userid", "", "teamid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You must specify a channel name.")

	resp, err = p.SetChannel("userid", "~", "teamid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You must specify a channel name.")

	resp, err = p.SetChannel("userid", "town-square", "teamid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Channel set to mock.")

	resp, err = p.SetChannel("userid", "~town-square", "teamid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Channel set to mock.")

	// Fails.
	p = initTestPlugin(t, "team", "fail")
	assert.Nil(t, p.OnActivate())

	resp, err = p.SetChannel("userid", "town-square", "teamid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "\"town-square\" isn't a valid channel, use one that exists.")

	resp, err = p.SetChannel("userid", "~town-square", "teamid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "\"town-square\" isn't a valid channel, use one that exists.")
}

// TestSetInterval - test the SetInterval function.
func TestSetInterval(t *testing.T) {
	// Regular user testing.
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err := p.SetInterval("userid", "15")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Only admins can set the interval.")

	// Admin testing.
	p = initTestPlugin(t, "team", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err = p.SetInterval("userid", "")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You have to specify an interval in minutes, >= 15.")

	resp, err = p.SetInterval("userid", "cat")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You have to specify an interval in minutes, >= 15.")

	resp, err = p.SetInterval("userid", "5")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You can't set an Interval less than 15 minutes, it's annoying.")

	resp, err = p.SetInterval("userid", "15")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Interval set to 15 minutes.")

	resp, err = p.SetInterval("userid", "60")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Interval set to 60 minutes.")

	resp, err = p.SetInterval("userid", "10081")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You can't set the Interval to more than a week, that's excessive.")
}

// TestShowHelpAdmin - Test the ShowHelp function.
func TestShowHelpAdmin(t *testing.T) {
	p := initTestPlugin(t, "system", "mock")

	resp, err := p.ShowHelp("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, strings.Join([]string{helpText, adminHelpText}, "\n\n"))
}

// -----------------------------------------------------------------------------
// Quotebot commands
// -----------------------------------------------------------------------------

// TestAddQuote - Test the AddQuote function.
func TestAddQuote(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")
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

// TestShowHelp - Test the ShowHelp function.
func TestShowHelp(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")

	resp, err := p.ShowHelp("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, helpText)
}

// TestShowInfo - Test the ShowInfo function.
func TestShowInfo(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err := p.ShowInfo("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You are a User. Quotebot knows 0 quotes. Monitoring mock for activity every 15 minutes.")

	p = initTestPlugin(t, "normal", "fail")
	assert.Nil(t, p.OnActivate())

	resp, err = p.ShowInfo("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text,
		"You are a User. Quotebot knows 0 quotes. Monitoring a non-existent channel. An Admin should fix that.")

	p = initTestPlugin(t, "channel", "mock")
	assert.Nil(t, p.OnActivate())

	resp, err = p.ShowInfo("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You are an Admin. Quotebot knows 0 quotes. Monitoring mock for activity every 15 minutes.")

	p = initTestPlugin(t, "channel", "fail")
	assert.Nil(t, p.OnActivate())

	resp, err = p.ShowInfo("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "You are an Admin. Quotebot knows 0 quotes. Monitoring a non-existent channel. An Admin should fix that.")
}

// TestShowQuote - Test the ShowQuote function.
func TestShowQuote(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())
	assert.EqualValues(t, len(p.quotes), 0)

	resp, err := p.ShowQuote("userid", "foo") // "" calls ShowRandom() instead of ShowQuote().
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, helpText)

	resp, err = p.ShowQuote("userid", "1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = p.AddQuote("quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)

	resp, err = p.ShowQuote("userid", "1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_IN_CHANNEL)
	assert.EqualValues(t, resp.Text, "> quote 1")

	resp, err = p.ShowQuote("userid", "2")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "Unable to show quote 2, it doesn't exist yet. There are 1 quotes on file.")
}

// TestShowRandom - Test the ShowRandom function.
func TestShowRandom(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())
	assert.EqualValues(t, len(p.quotes), 0)

	resp, err := p.ShowRandom("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_EPHEMERAL)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = p.AddQuote("quote 1")
	assert.NotNil(t, resp)
	assert.Nil(t, err)

	resp, err = p.ShowRandom("userid")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.ResponseType, model.COMMAND_RESPONSE_TYPE_IN_CHANNEL)
	assert.EqualValues(t, resp.Text, "> quote 1")
}
