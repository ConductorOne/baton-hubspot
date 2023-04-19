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

type accountResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (acc *accountResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return acc.resourceType
}

// Create a new connector resource for an HubSpot account.
func accountResource(ctx context.Context, account *hubspot.Account, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"account_id":   account.Id,
		"account_type": account.Type,
	}

	resource, err := rs.NewAppResource(
		fmt.Sprint(account.Id),
		resourceTypeAccount,
		account.Id,
		[]rs.AppTraitOption{
			rs.WithAppProfile(profile),
		},
		rs.WithParentResourceID(parentResourceID),
		rs.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: resourceTypeUser.Id},
			&v2.ChildResourceType{ResourceTypeId: resourceTypeTeam.Id},
		),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (acc *accountResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	account, err := acc.client.GetAccount(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to list account: %w", err)
	}

	var rv []*v2.Resource
	accountCopy := account
	ar, err := accountResource(ctx, &accountCopy, parentId)
	if err != nil {
		return nil, "", nil, err
	}
	rv = append(rv, ar)

	return rv, "", nil, nil
}

func (acc *accountResourceType) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	// fetch all available user roles
	roles, _ := acc.client.GetRoles(ctx)
	if roles == nil {
		// do not return user entitlements when account does not support roles
		return nil, "", nil, nil
	}

	var rv []*v2.Entitlement

	for _, role := range roles {
		assignmentOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeUser),
			ent.WithDisplayName(fmt.Sprintf("%s Acc %s", resource.DisplayName, titleCaser.String(role.Name))),
			ent.WithDescription(fmt.Sprintf("Account %s role in HubSpot", resource.DisplayName)),
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

func (acc *accountResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	// fetch all available user roles
	roles, _ := acc.client.GetRoles(ctx)
	if roles == nil {
		// do not return user grants when account does not support roles
		return nil, "", nil, nil
	}

	// parse the roleIds from the users
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextToken, err := acc.client.GetUsers(
		ctx,
		hubspot.GetUsersVars{Limit: ResourcesPageSize, After: bag.PageToken()},
	)
	if err != nil {
		return nil, "", nil, err
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Grant
	for _, user := range users {
		for _, roleId := range user.RoleIds {
			role, err := findRole(roleId, roles)
			if err != nil {
				continue
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
					role.Name,
					u.Id,
				),
			)
		}
	}

	return rv, pageToken, nil, nil
}

func accountBuilder(client *hubspot.Client) *accountResourceType {
	return &accountResourceType{
		resourceType: resourceTypeAccount,
		client:       client,
	}
}
