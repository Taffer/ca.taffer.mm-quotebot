package main

import (
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-server/plugin"
)

func main() {
	quotebot := &QuotebotPlugin{}

	rand.Seed(time.Now().UnixNano())
	plugin.ClientMain(quotebot)
}
