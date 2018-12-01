package main

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
// Tests - Quotebot functions
// -----------------------------------------------------------------------------

// TestIsAdmin - Test whether I can tell if you're an admin or not.
func TestIsAdmin(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")
	assert.False(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "channel", "mock")
	assert.True(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "team", "mock")
	assert.True(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "system", "mock")
	assert.True(t, p.IsAdmin("userid"))

	p = initTestPlugin(t, "everything", "mock")
	assert.True(t, p.IsAdmin("userid"))
}

// TestLoadQuotes - Test the LoadQuotes function.
func TestLoadQuotes(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")
	err := p.LoadQuotes()
	assert.Nil(t, err)
	assert.EqualValues(t, len(p.quotes), 0)

	api := initAPI(t, "normal", "mock", []byte(`["quote 1", "quote 2"]`))
	p.SetAPI(api)
	err = p.LoadQuotes()
	assert.Nil(t, err)
	assert.EqualValues(t, len(p.quotes), 2)
}

// TestNewError - Test the NewError function.
func TestNewError(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")

	err := p.NewError("message", "details", "where")
	assert.EqualValues(t, err.Message, "message")
	assert.EqualValues(t, err.DetailedError, "details")
	assert.EqualValues(t, err.Where, "QuotebotPlugin.where")
}

// TestNewResponse - Test the NewResponse function.
func TestNewResponse(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")

	resp := p.NewResponse("type", "string")
	assert.EqualValues(t, resp.ResponseType, "type")
	assert.EqualValues(t, resp.Text, "string")
}

// // TestPostRandom - Test the PostRandom function.
// func TestPostRandom(t *testing.T) {
// }

// TestSaveQuotes - Test the SaveQuotes function.
// This test is weak. Not sure how to mock this to test it in a useful way,
// it's a fairly trivial function...
func TestSaveQuotes(t *testing.T) {
	p := initTestPlugin(t, "normal", "mock")
	assert.Nil(t, p.OnActivate())
	assert.EqualValues(t, len(p.quotes), 0)
}
