package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-hubspot/pkg/hubspot"
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

func (t *teamResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return t.resourceType
}

// Create a new connector resource for an HubSpot Team.
func teamResource(ctx context.Context, team *hubspot.Team, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	userIdsTotal := len(team.UserIds) + len(team.SecondaryUserIds)
	profile := map[string]interface{}{
		"team_id":   team.Id,
		"team_name": team.Name,
	}

	if userIdsTotal > 0 {
		userIds := make([]string, userIdsTotal)

		copy(userIds, team.UserIds)
		copy(userIds, team.SecondaryUserIds)

		profile["team_users"] = strings.Join(userIds, ",")
	}

	resource, err := rs.NewGroupResource(
		team.Name,
		resourceTypeTeam,
		team.Id,
		[]rs.GroupTraitOption{rs.WithGroupProfile(profile)},
		rs.WithParentResourceID(parentResourceID),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (t *teamResourceType) List(ctx context.Context, parentId *v2.ResourceId, _ *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	teams, annotations, err := t.client.GetTeams(ctx)
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

	return rv, "", annotations, nil
}

func (t *teamResourceType) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
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

func (t *teamResourceType) Grants(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	teamTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	userIdsString, ok := rs.GetProfileStringValue(teamTrait.Profile, "team_users")
	if !ok {
		return nil, "", nil, nil
	}

	userIds := strings.Split(userIdsString, ",")

	// create membership grants
	var rv []*v2.Grant
	for _, id := range userIds {
		user, _, err := t.client.GetUser(ctx, id)
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
