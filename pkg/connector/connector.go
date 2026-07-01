package connector

import (
	"context"

	"github.com/conductorone/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HubSpot struct {
	client     *hubspot.Client
	userStatus bool
}

func (hs *HubSpot) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncerV2 {
	return []connectorbuilder.ResourceSyncerV2{
		accountBuilder(hs.client),
		teamBuilder(hs.client),
		userBuilder(hs.client, hs.userStatus),
		roleBuilder(hs.client),
	}
}

// Metadata returns metadata about the connector.
func (hs *HubSpot) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return v2.ConnectorMetadata_builder{
		DisplayName:           "HubSpot",
		AccountCreationSchema: accountCreationSchema(),
	}.Build(), nil
}

func accountCreationSchema() *v2.ConnectorAccountCreationSchema {
	strField := func(displayName, description, placeholder string, order int32) *v2.ConnectorAccountCreationSchema_Field {
		return v2.ConnectorAccountCreationSchema_Field_builder{
			DisplayName: displayName,
			Description: description,
			Placeholder: placeholder,
			Required:    false,
			Order:       order,
			StringField: &v2.ConnectorAccountCreationSchema_StringField{},
		}.Build()
	}
	defaultTrue := true
	return v2.ConnectorAccountCreationSchema_builder{
		FieldMap: map[string]*v2.ConnectorAccountCreationSchema_Field{
			profileFieldEmail: v2.ConnectorAccountCreationSchema_Field_builder{
				DisplayName: "Email",
				Description: "The email address to invite to the HubSpot portal.",
				Placeholder: "jane@example.com",
				Required:    true,
				Order:       1,
				StringField: &v2.ConnectorAccountCreationSchema_StringField{},
			}.Build(),
			profileFieldFirstName:     strField("First Name", "The user's first name.", "Jane", 2),
			profileFieldLastName:      strField("Last Name", "The user's last name.", "Doe", 3),
			profileFieldRoleID:        strField("Role ID", "The ID of the role to assign to the user.", "", 4),
			profileFieldPrimaryTeamID: strField("Primary Team ID", "The ID of the user's primary team.", "", 5),
			profileFieldSecondaryTeamIDs: v2.ConnectorAccountCreationSchema_Field_builder{
				DisplayName:     "Secondary Team IDs",
				Description:     "IDs of additional teams to assign to the user.",
				Required:        false,
				Order:           6,
				StringListField: &v2.ConnectorAccountCreationSchema_StringListField{},
			}.Build(),
			profileFieldSendWelcomeEmail: v2.ConnectorAccountCreationSchema_Field_builder{
				DisplayName: "Send Welcome Email",
				Description: "Send a welcome email to the new user upon invitation.",
				Required:    false,
				Order:       7,
				BoolField: v2.ConnectorAccountCreationSchema_BoolField_builder{
					DefaultValue: &defaultTrue,
				}.Build(),
			}.Build(),
		},
	}.Build()
}

// Validate hits the HubSpot API to verify that the credentials are valid.
func (hs *HubSpot) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, annotations, err := hs.client.GetAccount(ctx)

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Provided Access Token is invalid")
	}

	return annotations, nil
}

// New returns the HubSpot connector.
func New(ctx context.Context, accessToken string, userStatus bool, baseURL string) (*HubSpot, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))

	if err != nil {
		return nil, err
	}

	client, err := hubspot.NewClient(accessToken, httpClient, baseURL)
	if err != nil {
		return nil, err
	}

	return &HubSpot{
		client:     client,
		userStatus: userStatus,
	}, nil
}
