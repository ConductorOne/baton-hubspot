package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

func capabilityPermissions(perms ...string) *v2.CapabilityPermissions {
	permissions := make([]*v2.CapabilityPermission, 0, len(perms))
	for _, p := range perms {
		permissions = append(permissions, v2.CapabilityPermission_builder{Permission: p}.Build())
	}
	return v2.CapabilityPermissions_builder{Permissions: permissions}.Build()
}

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: annotations.New(
			&v2.SkipEntitlementsAndGrants{},
			capabilityPermissions(
				"settings.users",              // sync user list
				"account-info.security.read",  // last login
				"crm.objects.users.read",      // user status (optional, --user-status flag)
				"settings.users.write",        // invite, delete, disable action
			),
		),
	}
	resourceTypeTeam = &v2.ResourceType{
		Id:          "team",
		DisplayName: "Team",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
		Annotations: annotations.New(
			capabilityPermissions(
				"settings.users.teams", // sync team list
				"settings.users",       // sync team members
				"settings.users.write", // grant/revoke team membership
			),
		),
	}
	resourceTypeAccount = &v2.ResourceType{
		Id:          "account",
		DisplayName: "Account",
		Annotations: annotations.New(
			capabilityPermissions(
				"account-info.security.read", // sync account details
			),
		),
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
		Annotations: annotations.New(
			capabilityPermissions(
				"settings.users",       // sync role list and assignments
				"settings.users.write", // grant/revoke role assignment
			),
		),
	}
)
