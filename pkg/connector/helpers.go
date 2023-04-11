package connector

import (
	"fmt"

	"github.com/ConductorOne/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var ResourcesPageSize = 50
var titleCaser = cases.Title(language.English)

func parsePageToken(i string, resourceID *v2.ResourceId) (*pagination.Bag, error) {
	b := &pagination.Bag{}
	err := b.Unmarshal(i)
	if err != nil {
		return nil, err
	}

	if b.Current() == nil {
		b.Push(pagination.PageState{
			ResourceTypeID: resourceID.ResourceType,
			ResourceID:     resourceID.Resource,
		})
	}

	return b, nil
}

func findRole(roleId string, roles []hubspot.Role) (hubspot.Role, error) {
	for _, role := range roles {
		if role.Id == roleId {
			return role, nil
		}
	}

	return hubspot.Role{}, fmt.Errorf("role id %s not found in %v", roleId, roles)
}
