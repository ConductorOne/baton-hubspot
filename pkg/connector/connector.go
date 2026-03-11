package connector

import (
	"context"
	"sync"

	"github.com/conductorone/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// deletedUsersSet tracks deactivated HubSpot users across resource types.
// It is populated during user listing and consulted during grant creation
// to skip grants for disabled users.
type deletedUsersSet struct {
	mu  sync.Mutex
	set map[string]bool
}

func (d *deletedUsersSet) Add(ids []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.set == nil {
		d.set = make(map[string]bool)
	}
	for _, id := range ids {
		d.set[id] = true
	}
}

func (d *deletedUsersSet) Contains(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.set[id]
}

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: annotationsForUserResourceType(),
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
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
	}
)

type HubSpot struct {
	client      *hubspot.Client
	userStatus  bool
	deletedUsers *deletedUsersSet
}

func (hs *HubSpot) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		accountBuilder(hs.client, hs.deletedUsers),
		teamBuilder(hs.client, hs.deletedUsers),
		userBuilder(hs.client, hs.userStatus, hs.deletedUsers),
		roleBuilder(hs.client, hs.deletedUsers),
	}
}

// Metadata returns metadata about the connector.
func (hs *HubSpot) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "HubSpot",
	}, nil
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
func New(ctx context.Context, accessToken string, userStatus bool) (*HubSpot, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))

	if err != nil {
		return nil, err
	}

	return &HubSpot{
		client:       hubspot.NewClient(accessToken, httpClient),
		userStatus:   userStatus,
		deletedUsers: &deletedUsersSet{},
	}, nil
}
