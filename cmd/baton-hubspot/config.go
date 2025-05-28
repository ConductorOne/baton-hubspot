package main

import (
	"github.com/conductorone/baton-sdk/pkg/field"

	"github.com/spf13/viper"
)

var (
	TokenField = field.StringField(
		"token",
		field.WithDescription("The HubSpot personal access token used to connect to the HubSpot API. ($BATON_TOKEN)"),
		field.WithRequired(true),
	)
	UserStatusField = field.BoolField(
		"user-status",
		field.WithDescription("Enables user status syncing. WARNING: Additional token scope needed: 'crm.objects.users.read'. ($BATON_USER_STATUS)"),
		field.WithDefaultValue(false),
	)
	// ConfigurationFields defines the external configuration required for the
	// connector to run. Note: these fields can be marked as optional or
	// required.
	ConfigurationFields = []field.SchemaField{TokenField, UserStatusField}
)

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(v *viper.Viper) error {
	return nil
}
