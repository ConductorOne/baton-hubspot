package connector

import (
	"github.com/conductorone/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var ResourcesPageSize = 50
var titleCaser = cases.Title(language.English)

func annotationsForUserResourceType() annotations.Annotations {
	annos := annotations.Annotations{}
	annos.Update(&v2.SkipEntitlementsAndGrants{})
	return annos
}

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

func filterUsersByRole(id string, users []hubspot.User) []hubspot.User {
	var filteredUsers []hubspot.User

	for _, user := range users {
		for _, roleId := range user.RoleIds {
			if roleId == id {
				filteredUsers = append(filteredUsers, user)
				break
			}
		}
	}

	return filteredUsers
}

func containsTeam(tIds []string, targetTeam string) bool {
	for _, id := range tIds {
		if id == targetTeam {
			return true
		}
	}

	return false
}

func removeTeam(tIds []string, targetTeam string) []string {
	tv := make([]string, 0, len(tIds))

	for _, id := range tIds {
		if id != targetTeam {
			tv = append(tv, id)
		}
	}

	return tv
}
