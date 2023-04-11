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

type userResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (o *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for an HubSpot user.
func userResource(ctx context.Context, user *hubspot.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	roleIds := make([]string, len(user.RoleIds))
	copy(roleIds, user.RoleIds)

	profile := map[string]interface{}{
		"login":         user.Email,
		"user_id":       user.Id,
		"user_role_ids": strings.Join(roleIds, ","),
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithEmail(user.Email, true),
		rs.WithStatus(v2.UserTrait_Status_STATUS_UNSPECIFIED),
	}

	resource, err := rs.NewUserResource(
		user.Email, // email as a name
		resourceTypeUser,
		user.Id,
		userTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (o *userResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextToken, err := o.client.GetUsers(
		ctx,
		hubspot.GetUsersVars{Limit: ResourcesPageSize, After: bag.PageToken()},
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to list users: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, user := range users {
		userCopy := user
		ur, err := userResource(ctx, &userCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	return rv, pageToken, nil, nil
}

func (o *userResourceType) Entitlements(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	// fetch all available user roles
	roles, err := o.client.GetRoles(ctx)
	if err != nil {
		// return empty list of entitlements as account does not support roles
		return nil, "", nil, nil
	}

	// parse the roleIds from the resource
	userTrait, err := rs.GetUserTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	userRoleIdsString, ok := rs.GetProfileStringValue(userTrait.Profile, "user_role_ids")
	if !ok {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to get user_role_ids from user profile")
	}

	userRoleIds := strings.Split(userRoleIdsString, ",")

	var rv []*v2.Entitlement
	for _, role := range roles {
		_, err := find(role.Id, userRoleIds)
		if err != nil {
			continue
		}

		assignmentOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeUser),
			ent.WithDisplayName(fmt.Sprintf("%s User %s", resource.DisplayName, titleCaser.String(role.Name))),
			ent.WithDescription(fmt.Sprintf("Permission access for a user %s in HubSpot", resource.DisplayName)),
		}

		// create the entitlement
		rv = append(rv, ent.NewPermissionEntitlement(
			resource,
			role.Name,
			assignmentOptions...,
		))
	}

	return rv, "", nil, nil
}

func (o *userResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	// fetch all available user roles
	roles, err := o.client.GetRoles(ctx)
	if err != nil {
		// return empty list of entitlements as account does not support roles
		return nil, "", nil, nil
	}

	// parse the roleIds from the resource
	userTrait, err := rs.GetUserTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	userRoleIdsString, ok := rs.GetProfileStringValue(userTrait.Profile, "user_role_ids")
	if !ok {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to get user_role_ids from user profile")
	}

	userRoleIds := strings.Split(userRoleIdsString, ",")

	var rv []*v2.Grant
	for _, role := range roles {
		_, err := find(role.Id, userRoleIds)
		if err != nil {
			continue
		}

		rv = append(
			rv,
			grant.NewGrant(
				resource,
				role.Name,
				resource.Id,
			),
		)
	}

	return rv, "", nil, nil
}

func userBuilder(client *hubspot.Client) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
	}
}
