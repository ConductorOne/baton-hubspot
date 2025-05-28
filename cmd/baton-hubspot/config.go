package main

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/spf13/cobra"
)

// config defines the external configuration required for the connector to run.
type config struct {
	cli.BaseConfig `mapstructure:",squash"` // Puts the base config options in the same place as the connector options
	AccessToken    string                   `mapstructure:"token"`
	UserStatus     bool                     `mapstructure:"user-status"`
}

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *config) error {
	if cfg.AccessToken == "" {
		return fmt.Errorf("access token is missing")
	}

	return nil
}

// cmdFlags sets the cmdFlags required for the connector.
func cmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("token", "", "The HubSpot personal access token used to connect to the HubSpot API. ($BATON_TOKEN)")
	cmd.PersistentFlags().Bool("user-status", false, "Whether to enable user status. WARNING: Additional token scope needed: 'crm.objects.users.read'. ($BATON_USER_STATUS)")
}
