package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

const (
	PageTypeDeleted   = "DELETED_USERS"
	PageTypeAllUsers  = "ALL_USERS"
	PageTypeCompleted = "COMPLETED"
)

type userResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
	userStatus   bool
	deletedUsers *deletedUsersSet
}

func (u *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return u.resourceType
}

// Create a new connector resource for an HubSpot user.
func (c *userResourceType) userResource(ctx context.Context, user *hubspot.User, parentResourceID *v2.ResourceId) (*v2.Resource, annotations.Annotations, error) {
	profile := map[string]interface{}{
		"login":   user.Email,
		"user_id": user.Id,
	}
	userState := v2.UserTrait_Status_STATUS_ENABLED
	if c.deletedUsers.Contains(user.Id) {
		userState = v2.UserTrait_Status_STATUS_DISABLED
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithEmail(user.Email, true),
		rs.WithStatus(userState),
	}

	lastLogin, annos, err := c.client.GetUserLastLogin(ctx, user.Id)
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == 403 {
			l := ctxzap.Extract(ctx)
			l.Warn("baton-hubspot: failed to get last login activity: permission denied", zap.String("user_id", user.Id), zap.Error(err))
		} else {
			return nil, annos, err
		}
	}
	if lastLogin != nil {
		userTraitOptions = append(userTraitOptions, rs.WithLastLogin(*lastLogin))
	}

	resource, err := rs.NewUserResource(
		user.Email, // email as a name
		resourceTypeUser,
		user.Id,
		userTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)

	if err != nil {
		return nil, nil, err
	}

	return resource, nil, nil
}

func (u *userResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	userPageToken, err := unmarshalUserPageToken(bag.PageToken())
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to unmarshal the token %w", err)
	}

	if userPageToken.Type == "" && !u.userStatus {
		userPageToken = &UsersPaginationToken{Page: "", Type: PageTypeAllUsers}
	}

	switch userPageToken.Type {
	case "", PageTypeDeleted:
		// Paginate over deleted users and populate deleted set.
		deletedIDs, nextToken, annotation, err := u.client.GetDeletedUsers(ctx,
			hubspot.GetUsersVars{Limit: ResourcesPageSize, After: userPageToken.Page},
		)
		if err != nil {
			return nil, "", nil, fmt.Errorf("hubspot-connector: failed to get deactivated users: %w", err)
		}
		u.deletedUsers.Add(deletedIDs)
		if nextToken != "" {
			parsedNextToken, err := parseUserPaginationToken(
				UsersPaginationToken{Page: nextToken, Type: PageTypeDeleted},
				bag,
			)
			if err != nil {
				return nil, "", nil, err
			}
			return nil, parsedNextToken, annotation, nil
		} else {
			// no more deleted users, start PageTypeAllUsers pagination
			parsedNextToken, err := parseUserPaginationToken(
				UsersPaginationToken{Page: "", Type: PageTypeAllUsers},
				bag,
			)
			if err != nil {
				return nil, "", nil, err
			}
			return nil, parsedNextToken, annotation, nil
		}
	case PageTypeAllUsers:
		users, nextToken, annotations, err := u.client.GetUsers(
			ctx,
			hubspot.GetUsersVars{Limit: ResourcesPageSize, After: userPageToken.Page},
		)
		if err != nil {
			return nil, "", nil, fmt.Errorf("hubspot-connector: failed to list users: %w", err)
		}
		paginationType := PageTypeAllUsers
		if nextToken == "" {
			paginationType = PageTypeCompleted
		}
		parsedNextToken, err := parseUserPaginationToken(
			UsersPaginationToken{Page: nextToken, Type: paginationType},
			bag,
		)
		if err != nil {
			return nil, "", nil, err
		}

		var rv []*v2.Resource
		for _, user := range users {
			userCopy := user
			ur, annos, err := u.userResource(ctx, &userCopy, parentId)
			if err != nil {
				return nil, "", annos, err
			}

			rv = append(rv, ur)
		}

		return rv, parsedNextToken, annotations, nil
	case PageTypeCompleted:
		return nil, "", nil, nil
	}
	return nil, "", nil, nil
}

func (u *userResourceType) Entitlements(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (u *userResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func userBuilder(client *hubspot.Client, userStatus bool, deletedUsers *deletedUsersSet) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
		userStatus:   userStatus,
		deletedUsers: deletedUsers,
	}
}
