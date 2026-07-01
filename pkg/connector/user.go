package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-hubspot/pkg/hubspot"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/session"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/types/sessions"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	PageTypeDeleted   = "DELETED_USERS"
	PageTypeAllUsers  = "ALL_USERS"
	PageTypeCompleted = "COMPLETED"

	deletedUsersSessionKey = "deleted_users"
)

type userResourceType struct {
	resourceType *v2.ResourceType
	client       *hubspot.Client
	userStatus   bool
}

func (u *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return u.resourceType
}

// userResource creates a new connector resource for a HubSpot user.
func (c *userResourceType) userResource(ctx context.Context, user *hubspot.User, parentResourceID *v2.ResourceId, deletedSet map[string]bool) (*v2.Resource, annotations.Annotations, error) {
	profile := map[string]interface{}{
		"login":   user.Email,
		"user_id": user.Id,
	}
	userState := v2.UserTrait_Status_STATUS_ENABLED
	if deletedSet[user.Id] {
		userState = v2.UserTrait_Status_STATUS_DISABLED
	}

	userTraitOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithEmail(user.Email, true),
		rs.WithStatus(userState),
	}

	lastLogin, annos, err := c.client.GetUserLastLogin(ctx, user.Id)
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.PermissionDenied {
			l := ctxzap.Extract(ctx)
			l.Debug("baton-hubspot: failed to get last login activity: permission denied", zap.String("user_id", user.Id), zap.Error(err))
		} else {
			return nil, annos, err
		}
	}
	if lastLogin != nil {
		userTraitOptions = append(userTraitOptions, rs.WithLastLogin(*lastLogin))
	}

	resource, err := rs.NewUserResource(
		user.Email,
		resourceTypeUser,
		user.Id,
		userTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, annos, err
	}

	return resource, annos, nil
}

func (u *userResourceType) List(ctx context.Context, parentId *v2.ResourceId, opts rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	if parentId == nil {
		return nil, nil, nil
	}
	bag, err := parsePageToken(opts.PageToken.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, nil, err
	}

	userPageToken, err := unmarshalUserPageToken(bag.PageToken())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal the token %w", err)
	}

	if userPageToken.Type == "" && !u.userStatus {
		userPageToken = &UsersPaginationToken{Page: "", Type: PageTypeAllUsers}
	}

	switch userPageToken.Type {
	case "", PageTypeDeleted:
		deletedIDs, nextToken, annos, err := u.client.GetDeletedUsers(ctx,
			hubspot.GetUsersVars{Limit: ResourcesPageSize, After: userPageToken.Page},
		)
		if err != nil {
			return nil, nil, fmt.Errorf("hubspot-connector: failed to get deactivated users: %w", err)
		}

		deletedMap := make(map[string]bool, len(deletedIDs))
		for _, id := range deletedIDs {
			deletedMap[id] = true
		}
		if err := session.SetManyJSON(ctx, opts.Session, deletedMap, sessions.WithPrefix(deletedUsersSessionKey)); err != nil {
			return nil, nil, fmt.Errorf("hubspot-connector: failed to save deleted users to session: %w", err)
		}

		var nextPageType string
		var nextPage string
		if nextToken != "" {
			nextPageType = PageTypeDeleted
			nextPage = nextToken
		} else {
			nextPageType = PageTypeAllUsers
		}
		parsedNextToken, err := parseUserPaginationToken(
			UsersPaginationToken{Page: nextPage, Type: nextPageType},
			bag,
		)
		if err != nil {
			return nil, nil, err
		}
		return nil, &rs.SyncOpResults{NextPageToken: parsedNextToken, Annotations: annos}, nil

	case PageTypeAllUsers:
		users, nextToken, annos, err := u.client.GetUsers(
			ctx,
			hubspot.GetUsersVars{Limit: ResourcesPageSize, After: userPageToken.Page},
		)
		if err != nil {
			return nil, nil, fmt.Errorf("hubspot-connector: failed to list users: %w", err)
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
			return nil, nil, err
		}

		var deletedSet map[string]bool
		if u.userStatus {
			userIDs := make([]string, 0, len(users))
			for _, user := range users {
				userIDs = append(userIDs, user.Id)
			}
			deletedSet, err = session.GetManyJSON[bool](ctx, opts.Session, userIDs, sessions.WithPrefix(deletedUsersSessionKey))
			if err != nil {
				return nil, nil, fmt.Errorf("hubspot-connector: failed to get deleted users from session: %w", err)
			}
		}

		var rv []*v2.Resource
		for _, user := range users {
			userCopy := user
			ur, userAnnotations, err := u.userResource(ctx, &userCopy, parentId, deletedSet)
			if err != nil {
				return nil, &rs.SyncOpResults{Annotations: userAnnotations}, err
			}
			annos.Merge(userAnnotations...)
			rv = append(rv, ur)
		}

		return rv, &rs.SyncOpResults{NextPageToken: parsedNextToken, Annotations: annos}, nil

	case PageTypeCompleted:
		return nil, nil, nil
	}

	return nil, nil, nil
}

func (u *userResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

func (u *userResourceType) Grants(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

// CreateAccountCapabilityDetails advertises that HubSpot account creation is invitation-based;
// the invited user sets their own password, so no password is managed by the connector.
func (u *userResourceType) CreateAccountCapabilityDetails(_ context.Context) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return v2.CredentialDetailsAccountProvisioning_builder{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}.Build(), nil, nil
}

// CreateAccount invites a new user to the HubSpot portal by sending them an invitation email.
func (u *userResourceType) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	_ *v2.LocalCredentialOptions,
) (connectorbuilder.CreateAccountResponse, []*v2.PlaintextData, annotations.Annotations, error) {
	profile := accountInfo.GetProfile()
	if profile == nil {
		return nil, nil, nil, status.Error(codes.InvalidArgument, "baton-hubspot: account info has no profile")
	}
	fields := profile.GetFields()

	email := stringFromProfileField(fields, profileFieldEmail)
	if email == "" {
		return nil, nil, nil, status.Error(codes.InvalidArgument, "baton-hubspot: email field is required")
	}

	opts := hubspot.InviteUserOptions{
		SendWelcomeEmail: boolFromProfileField(fields, profileFieldSendWelcomeEmail, true),
		FirstName:        stringFromProfileField(fields, profileFieldFirstName),
		LastName:         stringFromProfileField(fields, profileFieldLastName),
		RoleID:           stringFromProfileField(fields, profileFieldRoleID),
		PrimaryTeamID:    stringFromProfileField(fields, profileFieldPrimaryTeamID),
		SecondaryTeamIDs: stringListFromProfileField(fields, profileFieldSecondaryTeamIDs),
	}

	user, annos, err := u.client.InviteUser(ctx, email, opts)
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.AlreadyExists {
			return v2.CreateAccountResponse_AlreadyExistsResult_builder{
				IsCreateAccountResult: true,
			}.Build(), nil, annos, nil
		}
		return nil, nil, nil, fmt.Errorf("baton-hubspot: failed to invite user: %w", err)
	}

	resource, resourceAnnotations, err := u.userResource(ctx, user, nil, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-hubspot: failed to build user resource: %w", err)
	}
	annos.Merge(resourceAnnotations...)

	return v2.CreateAccountResponse_SuccessResult_builder{
		IsCreateAccountResult: true,
		Resource:              resource,
	}.Build(), nil, annos, nil
}

// Delete removes a user from the HubSpot portal via the Settings API.
func (u *userResourceType) Delete(ctx context.Context, resourceId *v2.ResourceId, _ *v2.ResourceId) (annotations.Annotations, error) {
	annos, err := u.client.DeleteUser(ctx, resourceId.Resource)
	if err != nil {
		return nil, fmt.Errorf("baton-hubspot: failed to delete user: %w", err)
	}
	return annos, nil
}

func userBuilder(client *hubspot.Client, userStatus bool) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
		userStatus:   userStatus,
	}
}
