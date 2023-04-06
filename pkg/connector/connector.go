package connector

import (
	"context"

	"github.com/ConductorOne/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
	}
	resourceTypeTeam = &v2.ResourceType{
		Id:          "team",
		DisplayName: "Team",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
	}
	resourceTypeAccount = &v2.ResourceType{
		Id:          "account",
		DisplayName: "Account",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_APP,
		},
	}
)

type HubSpot struct {
	client *hubspot.Client
}

func (c *HubSpot) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(c.client),
		teamBuilder(c.client),
		accountBuilder(c.client),
	}
}

// Metadata returns metadata about the connector.
func (as *HubSpot) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "HubSpot",
	}, nil
}

// Validate hits the HubSpot API to verify that the credentials are valid.
func (as *HubSpot) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, err := as.client.GetAccount(ctx)

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Provided Access Token is invalid")
	}

	return nil, nil
}

// New returns the HubSpot connector.
func New(ctx context.Context, accessToken string) (*HubSpot, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))

	if err != nil {
		return nil, err
	}

	return &HubSpot{
		client: hubspot.NewClient(accessToken, httpClient),
	}, nil
}
