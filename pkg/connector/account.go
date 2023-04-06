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

type accountResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
}

func (o *accountResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
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
		[]rs.AppTraitOption{rs.WithAppProfile(profile)},
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (o *accountResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	account, err := o.client.GetAccount(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("hubspot-connector: failed to list account: %w", err)
	}

	var rv []*v2.Resource
	accountCopy := account
	acc, err := accountResource(ctx, &accountCopy, parentId)
	if err != nil {
		return nil, "", nil, err
	}
	rv = append(rv, acc)

	return rv, "", nil, nil
}

func (o *accountResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *accountResourceType) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func accountBuilder(client *hubspot.Client) *accountResourceType {
	return &accountResourceType{
		resourceType: resourceTypeAccount,
		client:       client,
	}
}
