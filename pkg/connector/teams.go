package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type teamResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (o *teamResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for an HubSpot team.
func teamResource(ctx context.Context, team *hubspot.Team, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"members_count": team.GetMembersCount(ctx),
	}

	resource, err := rs.NewGroupResource(
		team.Name,
		resourceTypeTeam,
		team.Id,
		[]rs.GroupTraitOption{rs.WithGroupProfile(profile)},
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (o *teamResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	teams, _, _, err := o.client.GetTeams(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to list teams: %w", err)
	}

	var rv []*v2.Resource
	for _, team := range teams {
		teamCopy := team
		ur, err := teamResource(ctx, &teamCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	return rv, "", nil, nil
}

func (o *teamResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *teamResourceType) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func teamBuilder(client *hubspot.Client) *teamResourceType {
	return &teamResourceType{
		resourceType: resourceTypeTeam,
		client:       client,
	}
}
