// Command dns01 is the official NGINX UI DNS-01 challenge plugin. It solves
// the ACME DNS-01 challenge through any of the DNS providers lego supports and
// speaks the nginx-ui plugin protocol on stdin and stdout.
package main

import (
	"context"

	sdk "github.com/nginxui/plugin-sdk-go"

	"github.com/nginxui/plugin-dns01/provider"
)

func main() {
	handler := provider.New()

	sdk.Serve(sdk.Plugin{
		DNS01: handler,
		Configure: func(_ context.Context, settings map[string]any) error {
			handler.SetSettings(provider.ParseSettings(settings))
			return nil
		},
	})
}
