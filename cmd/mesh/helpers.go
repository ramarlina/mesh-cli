package main

import (
	"github.com/ramarlina/mesh-cli/pkg/client"
	"github.com/ramarlina/mesh-cli/pkg/config"
	"github.com/ramarlina/mesh-cli/pkg/output"
	"github.com/ramarlina/mesh-cli/pkg/session"
)

// getOutputPrinter creates an output printer based on global flags
func getOutputPrinter() *output.Printer {
	format := output.FormatHuman
	if flagJSON {
		format = output.FormatJSON
	} else if flagRaw {
		format = output.FormatRaw
	}

	return output.New(format, flagQuiet, flagNoANSI)
}

// getClient creates an authenticated API client
func getClient() *client.Client {
	apiURL := config.GetAPIUrl()
	token := session.GetToken()
	return client.New(apiURL, client.WithToken(token))
}
