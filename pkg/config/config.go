package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	TokenField = field.StringField(
		"token",
		field.WithDisplayName("API client secret"),
		field.WithDescription("The HubSpot personal access token used to connect to the HubSpot API. ($BATON_TOKEN)"),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)
	UserStatusField = field.BoolField(
		"user-status",
		field.WithDisplayName("User status"),
		field.WithDescription("Enables user status syncing. WARNING: Additional token scope needed: 'crm.objects.users.read'. ($BATON_USER_STATUS)"),
		field.WithDefaultValue(false),
	)
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{TokenField, UserStatusField},
	field.WithConnectorDisplayName("HubSpot"),
	field.WithHelpUrl("/docs/baton/hubspot"),
	field.WithIconUrl("/static/app-icons/hubspot.svg"),
)
