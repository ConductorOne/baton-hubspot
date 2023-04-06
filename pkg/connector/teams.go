package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/ConductorOne/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const memberEntitlement = "member"

type teamResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (o *teamResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for an HubSpot team.
func teamResource(ctx context.Context, team *hubspot.Team, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	userIds := make([]string, len(team.UserIds)+len(team.SecondaryUserIds))

	copy(userIds, team.UserIds)
	copy(userIds, team.SecondaryUserIds)

	profile := map[string]interface{}{
		"team_id":    team.Id,
		"team_name":  team.Name,
		"team_users": strings.Join(userIds, ","),
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
	teams, err := o.client.GetTeams(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to list teams: %w", err)
	}

	var rv []*v2.Resource
	for _, team := range teams {
		teamCopy := team
		tResource, err := teamResource(ctx, &teamCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, tResource)
	}

	return rv, "", nil, nil
}

func (o *teamResourceType) Entitlements(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s Team %s", resource.DisplayName, memberEntitlement)),
		ent.WithDescription(fmt.Sprintf("Access to %s team in HubSpot", resource.DisplayName)),
	}

	// create membership entitlement
	rv = append(rv, ent.NewAssignmentEntitlement(
		resource,
		memberEntitlement,
		assignmentOptions...,
	))

	return rv, "", nil, nil
}

func (o *teamResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	teamTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	userIdsString, ok := rs.GetProfileStringValue(teamTrait.Profile, "team_users")
	if !ok {
		return nil, "", nil, fmt.Errorf("error fetching user ids from team profile")
	}

	userIds := strings.Split(userIdsString, ",")

	// create membership grants
	var rv []*v2.Grant
	for _, id := range userIds {
		user, err := o.client.GetUser(ctx, id)
		if err != nil {
			return nil, "", nil, err
		}

		userCopy := user
		u, err := userResource(ctx, &userCopy, nil)
		if err != nil {
			return nil, "", nil, err
		}

		rv = append(
			rv,
			grant.NewGrant(
				resource,
				memberEntitlement,
				u.Id,
			),
		)
	}

	return rv, "", nil, nil
}

func teamBuilder(client *hubspot.Client) *teamResourceType {
	return &teamResourceType{
		resourceType: resourceTypeTeam,
		client:       client,
	}
}
