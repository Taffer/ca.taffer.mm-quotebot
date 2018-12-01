package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// Tests - Plugin callbacks.
// -----------------------------------------------------------------------------

// TestOnActivate - Test the OnActivate callback.
func TestOnActivate(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")

	assert.False(t, p.active)
	p.OnActivate()
	assert.True(t, p.active)
}

// TestOnDeactivate - Test the OnDeactivate callback.
func TestOnDeactivate(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")

	assert.False(t, p.active)
	p.OnActivate()
	assert.True(t, p.active)
	p.OnDeactivate()
	assert.False(t, p.active)
}

// TestExecuteCommand - Test the ExecuteCommand callback.
func TestExecuteCommand(t *testing.T) {
	// Normal user commands.
	resp, err := runTestPluginCommand(t, "/quote", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = runTestPluginCommand(t, "/quote ", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = runTestPluginCommand(t, "/quote 1", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "There aren't any quotes yet.")

	resp, err = runTestPluginCommand(t, "/quote add", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Empty quote. Try adding a quote with some text.")

	resp, err = runTestPluginCommand(t, "/quote add some genius quote", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Added \"some genius quote\" as quote number 1.")

	resp, err = runTestPluginCommand(t, "/quote help", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, helpText)

	// Admin commands.
	resp, err = runTestPluginCommand(t, "/quote channel", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the channel.")

	resp, err = runTestPluginCommand(t, "/quote channel ~town-square", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the channel.")

	resp, err = runTestPluginCommand(t, "/quote delete", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can delete quotes.")

	resp, err = runTestPluginCommand(t, "/quote delete 1", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can delete quotes.")

	resp, err = runTestPluginCommand(t, "/quote interval", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the interval.")

	resp, err = runTestPluginCommand(t, "/quote interval 1", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can set the interval.")

	resp, err = runTestPluginCommand(t, "/quote list", "user", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Only admins can list the quotes.")
}

// TestExecuteCommandAdmin - Test the ExecuteCommand() triggers that require admin access.
func TestExecuteCommandAdmin(t *testing.T) {
	resp, err := runTestPluginCommand(t, "/quote channel", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You must specify a channel name.")

	resp, err = runTestPluginCommand(t, "/quote channel ~town-square", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Channel set to mock.")

	resp, err = runTestPluginCommand(t, "/quote delete", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "What quote? You have to specify a quote index.")

	// TODO: Test this with a list of actual quotes.
	resp, err = runTestPluginCommand(t, "/quote delete -1", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You can't delete quote -1, it doesn't exist.")

	resp, err = runTestPluginCommand(t, "/quote delete 1", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You can't delete quote 1, it doesn't exist.")

	resp, err = runTestPluginCommand(t, "/quote interval", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You have to specify an interval in minutes, >= 15.")

	resp, err = runTestPluginCommand(t, "/quote interval 1", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "You can't set an Interval less than 15 minutes, it's annoying.")

	resp, err = runTestPluginCommand(t, "/quote interval 120", "system", "mock")
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp.Text, "Interval set to 120 minutes.")

	resp, err = runTestPluginCommand(t, "/quote list", "system", "mock")
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
