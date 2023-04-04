package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
	}
)

type HubSpot struct {
	client *hubspot.Client
}

func (c *HubSpot) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(c.client),
	}
}

// Metadata returns metadata about the connector.
func (as *HubSpot) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}

// Validate hits the HubSpot API to validate that the API key passed has admin rights.
func (as *HubSpot) Validate(ctx context.Context) (annotations.Annotations, error) {
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
