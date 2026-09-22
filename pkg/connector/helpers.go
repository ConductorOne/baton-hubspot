package connector

import (
	"encoding/json"

	"github.com/conductorone/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/types/known/structpb"
)

var ResourcesPageSize = 1000

const (
	profileFieldEmail            = "email"
	profileFieldFirstName        = "first_name"
	profileFieldLastName         = "last_name"
	profileFieldRoleID           = "role_id"
	profileFieldPrimaryTeamID    = "primary_team_id"
	profileFieldSecondaryTeamIDs = "secondary_team_ids"
	profileFieldSendWelcomeEmail = "send_welcome_email"
)

type UsersPaginationToken struct {
	Page string `json:"page"`
	Type string `json:"type"`
}

func titleCase(s string) string {
	titleCaser := cases.Title(language.English)

	return titleCaser.String(s)
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

func parseUserPaginationToken(token UsersPaginationToken, bag *pagination.Bag) (string, error) {
	jsonToken, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	tokenStr := string(jsonToken)
	pageToken, err := bag.NextToken(tokenStr)
	if err != nil {
		return "", err
	}

	return pageToken, nil
}

func unmarshalUserPageToken(stringToken string) (*UsersPaginationToken, error) {
	var token UsersPaginationToken
	if stringToken == "" {
		return &token, nil
	}
	err := json.Unmarshal([]byte(stringToken), &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func filterUsersByRole(id string, users []hubspot.User) []hubspot.User {
	var filteredUsers []hubspot.User

	for _, user := range users {
		for _, roleId := range user.RoleIDs {
			if roleId == id {
				filteredUsers = append(filteredUsers, user)
				break
			}
		}
	}

	return filteredUsers
}

func filterUsersBySuperAdmin(users []hubspot.User) []hubspot.User {
	var superAdmins []hubspot.User

	for _, user := range users {
		if user.SuperAdmin {
			superAdmins = append(superAdmins, user)
		}
	}

	return superAdmins
}

func containsTeam(tIDs []string, targetTeam string) bool {
	for _, id := range tIDs {
		if id == targetTeam {
			return true
		}
	}

	return false
}


func getUserResourceId(userId string) *v2.ResourceId {
	return &v2.ResourceId{
		ResourceType: resourceTypeUser.Id,
		Resource:     userId,
	}
}

func stringFromProfileField(fields map[string]*structpb.Value, key string) string {
	if v, ok := fields[key]; ok {
		return v.GetStringValue()
	}
	return ""
}

func boolFromProfileField(fields map[string]*structpb.Value, key string, defaultVal bool) bool {
	v, ok := fields[key]
	if !ok {
		return defaultVal
	}
	if _, isBool := v.GetKind().(*structpb.Value_BoolValue); !isBool {
		return defaultVal
	}
	return v.GetBoolValue()
}

func stringListFromProfileField(fields map[string]*structpb.Value, key string) []string {
	v, ok := fields[key]
	if !ok {
		return nil
	}
	var result []string
	for _, item := range v.GetListValue().GetValues() {
		if s := item.GetStringValue(); s != "" {
			result = append(result, s)
		}
	}
	return result
}
