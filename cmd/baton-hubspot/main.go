package main

import (
	"context"

	cfg "github.com/conductorone/baton-hubspot/pkg/config"
	"github.com/conductorone/baton-hubspot/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
)

var version = "dev"

func main() {
	ctx := context.Background()

	config.RunConnector(
		ctx,
		"baton-hubspot",
		version,
		cfg.Config,
		getConnector,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.HubSpot{}),
		connectorrunner.WithSessionStoreEnabled(),
	)
}

func getConnector(ctx context.Context, hsc *cfg.Hubspot, _ *cli.ConnectorOpts) (connectorbuilder.ConnectorBuilderV2, []connectorbuilder.Opt, error) {
	c, err := connector.New(ctx, hsc.Token, hsc.UserStatus, hsc.BaseUrl)
	if err != nil {
		return nil, nil, err
	}
	return c, nil, nil
}
