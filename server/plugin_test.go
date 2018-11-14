package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// Utilities
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
// Tests - Plugin callbacks.
// -----------------------------------------------------------------------------

// TestOnActivate - Test the OnActivate callback.
func TestOnActivate(t *testing.T) {
}

// TestOnDeactivate - Test the OnDeactivate callback.
func TestOnDeactivate(t *testing.T) {
}

// TestExecuteCommand - Test the ExecuteCommand callback.
func TestExecuteCommand(t *testing.T) {
}

// TestMessageHasBeenPosted - Test the MessageHasBeenPosted callback.
func TestMessageHasBeenPosted(t *testing.T) {
}

// TestUserHasJoinedChannel - Test the UserHasJoinedChannel callback.
func TestUserHasJoinedChannel(t *testing.T) {
}

// TestUserHasLeftChannel - Test the UserHasLeftChannel callback.
func TestUserHasLeftChannel(t *testing.T) {
}

// -----------------------------------------------------------------------------
// Tests - Quotebot functions
// -----------------------------------------------------------------------------

// TestNewResponse - Test the NewResponse function.
func TestNewResponse(t *testing.T) {
}

// TestNewError - Test the NewError function.
func TestNewError(t *testing.T) {
}

// TestPostRandom - Test the PostRandom function.
func TestPostRandom(t *testing.T) {
}

// TestShowQuote - Test the ShowQuote function.
func TestShowQuote(t *testing.T) {
}

// TestShowRandom - Test the ShowRandom function.
func TestShowRandom(t *testing.T) {
}

// TestShowHelp - Test the ShowHelp function.
func TestShowHelp(t *testing.T) {
}

// TestAddQuote - Test the AddQuote function.
func TestAddQuote(t *testing.T) {
}

// TestListQuotes - Test the ListQuotes function.
func TestListQuotes(t *testing.T) {
}

// TestDeleteQuote - Test the DeleteQuote function.
func TestDeleteQuote(t *testing.T) {
}
