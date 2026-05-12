package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	roleMembership = "member"
	superAdminRole = "super_admin"
)

type roleResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (r *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return r.resourceType
}

// Create a new connector resource for an HubSpot user.
func roleResource(role *hubspot.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	displayName := titleCase(role.Name)
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

func (r *roleResourceType) List(ctx context.Context, parentId *v2.ResourceId, _ rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	if parentId == nil {
		return nil, nil, nil
	}

	roles, annos, _ := r.client.GetRoles(ctx)

	var rv []*v2.Resource
	for _, role := range roles {
		roleCopy := role

		rr, err := roleResource(&roleCopy, parentId)
		if err != nil {
			return nil, nil, err
		}

		rv = append(rv, rr)
	}

	// add concrete super admin role
	saRole := hubspot.NewRole(superAdminRole, "Super Admin")
	sar, err := roleResource(saRole, parentId)
	if err != nil {
		return nil, nil, err
	}

	rv = append(rv, sar)

	return rv, &rs.SyncOpResults{Annotations: annos}, nil
}

func (r *roleResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	var rv []*v2.Entitlement

	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s role", resource.DisplayName)),
		ent.WithDescription(fmt.Sprintf("%s role in HubSpot", resource.DisplayName)),
	}

	rv = append(rv, ent.NewAssignmentEntitlement(
		resource,
		roleMembership,
		assignmentOptions...,
	))

	return rv, nil, nil
}

func (r *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, opts rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	bag, err := parsePageToken(opts.PageToken.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, nil, err
	}

	users, nextToken, annos, err := r.client.GetUsers(ctx, hubspot.GetUsersVars{
		Limit: ResourcesPageSize,
		After: bag.PageToken(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("hubspot-connector: failed to list users: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, nil, err
	}

	roleTrait, err := rs.GetRoleTrait(resource)
	if err != nil {
		return nil, nil, err
	}

	roleId, ok := rs.GetProfileStringValue(roleTrait.Profile, "role_id")
	if !ok {
		return nil, nil, fmt.Errorf("hubspot-connector: error parsing role id from role profile")
	}

	var rv []*v2.Grant
	var filteredUsers []hubspot.User
	if roleId == superAdminRole {
		filteredUsers = filterUsersBySuperAdmin(users)
	} else {
		filteredUsers = filterUsersByRole(roleId, users)
	}

	for _, user := range filteredUsers {
		userResourceId := getUserResourceId(user.Id)
		rv = append(rv, grant.NewGrant(resource, roleMembership, userResourceId))
	}

	return rv, &rs.SyncOpResults{NextPageToken: pageToken, Annotations: annos}, nil
}

func (r *roleResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"hubspot-connector: only users can be granted role membership",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, nil, fmt.Errorf("hubspot-connector: only users can be granted role membership")
	}

	roleId := entitlement.Resource.Id.Resource

	if roleId == superAdminRole {
		return nil, nil, fmt.Errorf("hubspot-connector: super admin role can not be provisioned via API")
	}

	annos, err := r.client.UpdateUser(
		ctx,
		principal.Id.Resource,
		&hubspot.UpdateUserPayload{
			RoleId: roleId,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("hubspot-connector: failed to update user: %w", err)
	}

	return nil, annos, nil
}

func (r *roleResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	principal := grant.Principal

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"hubspot-connector: only users can have role membership revoked",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("hubspot-connector: only users can have role membership revoked")
	}

	roleId := grant.Entitlement.Resource.Id.Resource
	if roleId == superAdminRole {
		return nil, fmt.Errorf("hubspot-connector: super admin role can not be revoked via API")
	}

	annos, err := r.client.UpdateUser(
		ctx,
		principal.Id.Resource,
		&hubspot.UpdateUserPayload{},
	)
	if err != nil {
		return nil, fmt.Errorf("hubspot-connector: failed to update user: %w", err)
	}

	return annos, nil
}

func roleBuilder(client *hubspot.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
