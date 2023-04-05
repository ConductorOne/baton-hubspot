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
	profile := map[string]interface{}{
		"team_id":    team.Id,
		"team_name":  team.Name,
		"team_users": strings.Join(team.UserIds, ","),
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
		ur, err := teamResource(ctx, &teamCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
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

	// create an entitlement
	rv = append(rv, ent.NewAssignmentEntitlement(
		resource,
		memberEntitlement,
		assignmentOptions...,
	))

	return rv, "", nil, nil
}

func (o *teamResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	teamMembers, pageToken, err := o.GetTeamMembers(ctx, resource, token)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Grant
	for _, user := range teamMembers {
		u, err := userResource(ctx, &user, nil)

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

	return rv, pageToken, nil, nil
}

func (o *teamResourceType) GetTeamMembers(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]hubspot.User, string, error) {
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", err
	}

	users, nextToken, err := o.client.GetUsers(ctx, hubspot.GetUsersVars{
		Limit: ResourcesPageSize,
		After: bag.PageToken(),
	})
	if err != nil {
		return nil, "", fmt.Errorf("hubspot-connector: failed to fetch users: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", err
	}

	teamTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", err
	}

	userIdsPayload, ok := rs.GetProfileStringValue(teamTrait.Profile, "team_users")
	if !ok {
		return nil, "", fmt.Errorf("error fetching user ids from team profile")
	}

	userIds := strings.Split(userIdsPayload, ",")
	filterPresentUsers := func(user hubspot.User) bool {
		return includes(userIds, user.Id)
	}
	filteredUsers := filter(users, filterPresentUsers)

	return filteredUsers, pageToken, nil
}

func teamBuilder(client *hubspot.Client) *teamResourceType {
	return &teamResourceType{
		resourceType: resourceTypeTeam,
		client:       client,
	}
}
