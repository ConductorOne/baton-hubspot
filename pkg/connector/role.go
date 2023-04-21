package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type roleResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (r *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return r.resourceType
}

// Create a new connector resource for an HubSpot user.
func roleResource(ctx context.Context, role *hubspot.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	displayName := titleCaser.String(role.Name)
	profile := map[string]interface{}{
		"role_id":   role.Id,
		"role_name": displayName,
	}

	roleTraitOptions := []rs.RoleTraitOption{
		rs.WithRoleProfile(profile),
	}

	resource, err := rs.NewRoleResource(
		displayName,
		resourceTypeRole,
		role.Id,
		roleTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (r *roleResourceType) List(ctx context.Context, parentId *v2.ResourceId, _ *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	roles, annotations, _ := r.client.GetRoles(ctx)
	if roles == nil {
		// do not list user entitlements when account does not support roles
		return nil, "", annotations, nil
	}

	var rv []*v2.Resource
	for _, role := range roles {
		roleCopy := role

		rr, err := roleResource(ctx, &roleCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, rr)
	}

	return rv, "", annotations, nil
}

func (r *roleResourceType) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s role", resource.DisplayName)),
		ent.WithDescription(fmt.Sprintf("%s role in HubSpot", resource.DisplayName)),
	}

	// create membership entitlement
	rv = append(rv, ent.NewAssignmentEntitlement(
		resource,
		memberEntitlement,
		assignmentOptions...,
	))

	return rv, "", nil, nil
}

func (r *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextToken, annotations, err := r.client.GetUsers(ctx, hubspot.GetUsersVars{
		Limit: ResourcesPageSize,
		After: bag.PageToken(),
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to list users: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	// Parse the role id from the role profile
	roleTrait, err := rs.GetRoleTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	roleId, ok := rs.GetProfileStringValue(roleTrait.Profile, "role_id")
	if !ok {
		return nil, "", nil, fmt.Errorf("hubspot-connector: error parsing role id from role profile")
	}

	var rv []*v2.Grant
	for _, user := range filterUsersByRole(roleId, users) {
		userCopy := user
		ur, err := userResource(ctx, &userCopy, nil)
		if err != nil {
			return nil, "", nil, err
		}

		rv = append(rv, grant.NewGrant(
			resource,
			memberEntitlement,
			ur.Id,
		))
	}

	return rv, pageToken, annotations, nil
}

func roleBuilder(client *hubspot.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
