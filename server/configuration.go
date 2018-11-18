package main

import (
	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	postDelta   float64 // Minutes between posting random quotations.
	postChannel string  // Channel to post random quotations to.

	quotes []string // All of the quotes we know.
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *QuotebotPlugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *QuotebotPlugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		panic("setConfiguration called with the existing configuration.")
	}

	p.configuration = configuration
}

func (p *QuotebotPlugin) loadConfiguration(configuration *configuration) error {
	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "Failed to load plugin configuration.")
	}

	return nil
}

func (p *QuotebotPlugin) saveConfiguration(configuration *configuration) error {
	// Save the configuration.
	if err := p.API.SavePluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "Failed to save plugin configuration.")
	}

	return nil
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *QuotebotPlugin) OnConfigurationChange() error {
	configuration := new(configuration)
	err := p.loadConfiguration(configuration)
	if err == nil {
		p.setConfiguration(configuration)
	}

	return nil
}
