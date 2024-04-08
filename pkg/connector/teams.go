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
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	primaryMemberEntitlement   = "primary-member"
	secondaryMemberEntitlement = "secondary-member"
)

type teamResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (t *teamResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return t.resourceType
}

// Create a new connector resource for an HubSpot Team.
func teamResource(ctx context.Context, team *hubspot.Team, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"team_id":   team.Id,
		"team_name": team.Name,
	}

	if len(team.UserIDs) > 0 {
		profile["team_primary_users"] = strings.Join(team.UserIDs, ",")
	}

	if len(team.SecondaryUserIDs) > 0 {
		profile["team_secondary_users"] = strings.Join(team.SecondaryUserIDs, ",")
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
	primaryAssignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s Team primary member", resource.DisplayName)),
		ent.WithDescription(fmt.Sprintf("Access to %s team in HubSpot", resource.DisplayName)),
	}
	secondaryAssignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s Team secondary member", resource.DisplayName)),
		ent.WithDescription(fmt.Sprintf("Access to %s team in HubSpot", resource.DisplayName)),
	}

	// create membership entitlements
	rv = append(
		rv,
		ent.NewAssignmentEntitlement(
			resource,
			primaryMemberEntitlement,
			primaryAssignmentOptions...,
		),
		ent.NewAssignmentEntitlement(
			resource,
			secondaryMemberEntitlement,
			secondaryAssignmentOptions...,
		),
	)

	return rv, "", nil, nil
}

func (t *teamResourceType) Grants(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	teamTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	var primaryUserIDs, secondaryUserIDs []string

	primaryUserIDsString, ok := rs.GetProfileStringValue(teamTrait.Profile, "team_primary_users")
	if ok {
		primaryUserIDs = strings.Split(primaryUserIDsString, ",")
	}

	secondaryUserIDsString, ok := rs.GetProfileStringValue(teamTrait.Profile, "team_secondary_users")
	if ok {
		secondaryUserIDs = strings.Split(secondaryUserIDsString, ",")
	}

	// create membership grants
	var rv []*v2.Grant
	for _, id := range primaryUserIDs {
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
				primaryMemberEntitlement,
				u.Id,
			),
		)
	}

	for _, id := range secondaryUserIDs {
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
				secondaryMemberEntitlement,
				u.Id,
			),
		)
	}

	return rv, "", nil, nil
}

func (t *teamResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"hubspot-connector: only users can be granted team membership",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("hubspot-connector: only users can be granted team membership")
	}

	teamId := entitlement.Resource.Id.Resource
	entitlementId := entitlement.Slug

	// need to check principal role - without specifying role, it will be removed
	user, _, err := t.client.GetUser(ctx, principal.Id.Resource)
	if err != nil {
		return nil, fmt.Errorf("hubspot-connector: failed to get user: %w", err)
	}

	// there is only one role supported so far
	var roleId string
	if len(user.RoleIDs) != 0 {
		roleId = user.RoleIDs[0]
	}

	var annos annotations.Annotations
	if entitlementId == primaryMemberEntitlement {
		if user.TeamId == teamId {
			return nil, fmt.Errorf("hubspot-connector: user is already a primary member of team %s", teamId)
		}

		annos, err = t.client.UpdateUser(
			ctx,
			principal.Id.Resource,
			&hubspot.UpdateUserPayload{
				RoleId:        roleId,
				PrimaryTeamId: teamId,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("hubspot-connector: failed to update user: %w", err)
		}
	} else if entitlementId == secondaryMemberEntitlement {
		if containsTeam(user.SecondaryTeamIDs, teamId) {
			return nil, fmt.Errorf("hubspot-connector: user is already a secondary member of team %s", teamId)
		}

		annos, err = t.client.UpdateUser(
			ctx,
			principal.Id.Resource,
			&hubspot.UpdateUserPayload{
				RoleId:           roleId,
				SecondaryTeamIDs: append(user.SecondaryTeamIDs, teamId),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("hubspot-connector: failed to update user: %w", err)
		}
	}

	return annos, nil
}

func (t *teamResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	principal := grant.Principal
	entitlement := grant.Entitlement

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"hubspot-connector: only users can have team membership revoked",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("hubspot-connector: only users can have team membership revoked")
	}

	teamId := entitlement.Resource.Id.Resource
	entitlementId := entitlement.Slug

	user, _, err := t.client.GetUser(ctx, principal.Id.Resource)
	if err != nil {
		return nil, fmt.Errorf("hubspot-connector: failed to get user: %w", err)
	}

	var roleId string
	if len(user.RoleIDs) != 0 {
		roleId = user.RoleIDs[0]
	}

	var annos annotations.Annotations
	if entitlementId == primaryMemberEntitlement {
		if user.TeamId != teamId {
			return nil, fmt.Errorf("hubspot-connector: user is not a primary member of team %s", teamId)
		}

		annos, err = t.client.UpdateUser(
			ctx,
			principal.Id.Resource,
			&hubspot.UpdateUserPayload{
				RoleId: roleId,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("hubspot-connector: failed to update user: %w", err)
		}
	} else if entitlementId == secondaryMemberEntitlement {
		if !containsTeam(user.SecondaryTeamIDs, teamId) {
			return nil, fmt.Errorf("hubspot-connector: user is not a secondary member of team %s", teamId)
		}

		updatedTeams := removeTeam(user.SecondaryTeamIDs, teamId)
		annos, err = t.client.UpdateUser(
			ctx,
			principal.Id.Resource,
			&hubspot.UpdateUserPayload{
				RoleId:           roleId,
				SecondaryTeamIDs: updatedTeams,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("hubspot-connector: failed to updated user: %w", err)
		}
	}

	return annos, nil
}

func teamBuilder(client *hubspot.Client) *teamResourceType {
	return &teamResourceType{
		resourceType: resourceTypeTeam,
		client:       client,
	}
}
