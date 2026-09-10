package hubspot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const DefaultBaseURL = "https://api.hubapi.com/"
const EqualOperator = "EQ"
const HSInternalUserId = "hs_internal_user_id"

type Client struct {
	*uhttp.BaseHttpClient
	accessToken string
	baseURL     *url.URL
}

func (c *Client) usersURL() string {
	return c.baseURL.JoinPath("settings/users/2026-03").String()
}

func (c *Client) userURL(userID string) string {
	return c.baseURL.JoinPath("settings/users/2026-03", userID).String()
}

func (c *Client) teamsURL() string {
	return c.baseURL.JoinPath("settings/users/2026-03/teams").String()
}

func (c *Client) rolesURL() string {
	return c.baseURL.JoinPath("settings/users/2026-03/roles").String()
}

func (c *Client) accountURL() string {
	return c.baseURL.JoinPath("account-info/v3/details").String()
}

func (c *Client) searchUserObjectURL() string {
	return c.baseURL.JoinPath("crm/v3/objects/users/search").String()
}

func (c *Client) accountLastLoginURL() string {
	return c.baseURL.JoinPath("account-info/v3/activity/login").String()
}

type UsersResponse struct {
	Results []User         `json:"results"`
	Paging  PaginationData `json:"paging"`
}

type AccountLoginResponse struct {
	Results []LoginActivity `json:"results,omitempty"`
	Paging  PaginationData  `json:"paging,omitempty"`
}

type LoginActivity struct {
	LoginAt   time.Time `json:"loginAt,omitempty"`
	Succeeded bool      `json:"loginSucceeded,omitempty"`
}

type GetUsersVars struct {
	Limit int    `json:"limit"`
	After string `json:"after"`
}

type TeamsResponse struct {
	Results []Team `json:"results"`
}

type RolesResponse struct {
	Results []Role `json:"results"`
}

type SearchUserObjectResponse struct {
	Results []UserObject   `json:"results"`
	Paging  PaginationData `json:"paging"`
}

type Filters struct {
	Filters []Filter `json:"filters,omitempty"`
}

type Filter struct {
	PropertieName string `json:"propertyName,omitempty"`
	Operator      string `json:"operator,omitempty"`
	Value         string `json:"value,omitempty"`
}

type SearchUserObjectPayload struct {
	FilterGroups []Filters `json:"filterGroups,omitempty"`
	Properties   []string  `json:"properties,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	After        string    `json:"after,omitempty"`
}

func NewClient(accessToken string, httpClient *http.Client, baseURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("baton-hubspot: invalid base URL: %w", err)
	}
	return &Client{
		accessToken:    accessToken,
		baseURL:        parsedBaseURL,
		BaseHttpClient: uhttp.NewBaseHttpClient(httpClient),
	}, nil
}

func setupPaginationQuery(query url.Values, limit int, after string) url.Values {
	// add limit
	if limit != 0 {
		query.Add("limit", strconv.Itoa(limit))
	}

	// add page reference
	if after != "" {
		query.Add("after", after)
	}

	return query
}

// GetUsers returns all users for a single workspace.
func (c *Client) GetUsers(ctx context.Context, getUsersVars GetUsersVars) ([]User, string, annotations.Annotations, error) {
	queryParams := setupPaginationQuery(url.Values{}, getUsersVars.Limit, getUsersVars.After)
	var userResponse UsersResponse

	annos, err := c.get(
		ctx,
		c.usersURL(),
		&userResponse,
		queryParams,
	)

	if err != nil {
		return nil, "", nil, err
	}

	if (userResponse.Paging != PaginationData{}) {
		return userResponse.Results, userResponse.Paging.Next.After, annos, nil
	}

	return userResponse.Results, "", annos, nil
}

// GetTeams returns all teams for a single account.
func (c *Client) GetTeams(ctx context.Context) ([]Team, annotations.Annotations, error) {
	var teamResponse TeamsResponse
	annos, err := c.get(
		ctx,
		c.teamsURL(),
		&teamResponse,
		nil,
	)

	if err != nil {
		return nil, nil, err
	}

	return teamResponse.Results, annos, nil
}

// GetAccount returns information about single account.
func (c *Client) GetAccount(ctx context.Context) (Account, annotations.Annotations, error) {
	var accountResponse Account
	annos, err := c.get(
		ctx,
		c.accountURL(),
		&accountResponse,
		nil,
	)

	if err != nil {
		return Account{}, nil, err
	}

	return accountResponse, annos, nil
}

// GetUser returns information about a single user.
func (c *Client) GetUser(ctx context.Context, userId string) (User, annotations.Annotations, error) {
	var userResponse User
	annos, err := c.get(
		ctx,
		c.userURL(userId),
		&userResponse,
		nil,
	)
	if err != nil {
		return User{}, nil, err
	}

	return userResponse, annos, nil
}

// GetRoles returns all roles under a single account.
func (c *Client) GetRoles(ctx context.Context) ([]Role, annotations.Annotations, error) {
	var rolesResponse RolesResponse
	annos, err := c.get(ctx, c.rolesURL(), &rolesResponse, nil)
	if err != nil {
		return nil, nil, err
	}

	return rolesResponse.Results, annos, nil
}

type UpdateUserPayload struct {
	RoleId           string    `json:"roleId,omitempty"`
	PrimaryTeamId    *string   `json:"primaryTeamId,omitempty"`
	SecondaryTeamIDs *[]string `json:"secondaryTeamIds,omitempty"`
}

// UpdateUser updates information about provided user.
func (c *Client) UpdateUser(ctx context.Context, userId string, payload *UpdateUserPayload) (annotations.Annotations, error) {
	annos, err := c.put(
		ctx,
		c.userURL(userId),
		payload,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return annos, nil
}

func (c *Client) GetDeletedUsers(ctx context.Context, pageOptions GetUsersVars) ([]string, string, annotations.Annotations, error) {
	userFilter := Filter{
		PropertieName: "hs_deactivated",
		Operator:      EqualOperator,
		Value:         "true",
	}
	filters := []Filters{{Filters: []Filter{userFilter}}}
	payload := SearchUserObjectPayload{
		FilterGroups: filters,
		Properties:   []string{"hs_deactivated", HSInternalUserId},
		Limit:        pageOptions.Limit,
		After:        pageOptions.After,
	}
	var res SearchUserObjectResponse
	annos, err := c.post(
		ctx,
		c.searchUserObjectURL(),
		payload,
		&res,
	)
	if err != nil {
		return nil, "", nil, err
	}
	var ids []string
	for _, user := range res.Results {
		ids = append(ids, user.Properties.UserId)
	}
	if (res.Paging != PaginationData{}) {
		return ids, res.Paging.Next.After, annos, nil
	}
	return ids, "", annos, nil
}

// DeleteUser removes a user from the HubSpot portal via the Settings API.
func (c *Client) DeleteUser(ctx context.Context, userId string) (annotations.Annotations, error) {
	annos, err := c.delete(ctx, c.userURL(userId), nil)
	if err != nil {
		return nil, err
	}
	return annos, nil
}

// InviteUser sends an invitation to the provided email address, creating a new HubSpot portal user.
// Returns the created user. The caller is responsible for handling 409 (user already exists).
func (c *Client) InviteUser(ctx context.Context, email string, opts InviteUserOptions) (*User, annotations.Annotations, error) {
	payload := userInvitePayload{
		Email:            email,
		SendWelcomeEmail: opts.SendWelcomeEmail,
		FirstName:        opts.FirstName,
		LastName:         opts.LastName,
		RoleID:           opts.RoleID,
		PrimaryTeamID:    opts.PrimaryTeamID,
		SecondaryTeamIDs: opts.SecondaryTeamIDs,
	}
	var user User
	annos, err := c.post(ctx, c.usersURL(), payload, &user)
	if err != nil {
		return nil, nil, err
	}
	return &user, annos, nil
}

// GetUserLastLogin returns the last login time for a user.
// The /account-info/v3/activity/login endpoint requires account-info.security.read scope.
func (c *Client) GetUserLastLogin(ctx context.Context, userId string) (*time.Time, annotations.Annotations, error) {
	queryParams := setupPaginationQuery(url.Values{}, 5, "")
	var accountLoginResponse AccountLoginResponse
	queryParams.Add("userId", userId)

	annos, err := c.get(
		ctx,
		c.accountLastLoginURL(),
		&accountLoginResponse,
		queryParams,
	)
	if err != nil {
		return nil, annos, err
	}

	for _, loginActivity := range accountLoginResponse.Results {
		if loginActivity.Succeeded {
			return &loginActivity.LoginAt, annos, nil
		}
	}

	return nil, annos, nil
}

func (c *Client) get(ctx context.Context, url string, resourceResponse interface{}, queryParams url.Values) (annotations.Annotations, error) {
	return c.doRequest(ctx, url, http.MethodGet, nil, resourceResponse, queryParams)
}

func (c *Client) put(ctx context.Context, url string, data interface{}, resourceResponse interface{}) (annotations.Annotations, error) {
	return c.doRequest(ctx, url, http.MethodPut, data, resourceResponse, nil)
}

func (c *Client) post(ctx context.Context, url string, data interface{}, resourceResponse interface{}) (annotations.Annotations, error) {
	return c.doRequest(ctx, url, http.MethodPost, data, resourceResponse, nil)
}

func (c *Client) delete(ctx context.Context, url string, queryParams url.Values) (annotations.Annotations, error) {
	return c.doRequest(ctx, url, http.MethodDelete, nil, nil, queryParams)
}

func (c *Client) doRequest(
	ctx context.Context,
	urlAddress string,
	method string,
	data interface{},
	resourceResponse interface{},
	queryParams url.Values,
) (annotations.Annotations, error) {
	parsedURL, err := url.Parse(urlAddress)
	if err != nil {
		return nil, err
	}
	if queryParams != nil {
		parsedURL.RawQuery = queryParams.Encode()
	}

	reqOptions := []uhttp.RequestOption{
		uhttp.WithBearerToken(c.accessToken),
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithContentTypeJSONHeader(),
	}
	if data != nil {
		reqOptions = append(reqOptions, uhttp.WithJSONBody(data))
	}

	req, err := c.NewRequest(ctx, method, parsedURL, reqOptions...)
	if err != nil {
		return nil, err
	}

	var doOptions []uhttp.DoOption
	if resourceResponse != nil {
		doOptions = append(doOptions, uhttp.WithJSONResponse(resourceResponse))
	}

	resp, err := c.Do(req, doOptions...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rateLimitData, err := extractRateLimitData(resp)
	if err != nil {
		return nil, err
	}

	annos := annotations.Annotations{}
	annos.WithRateLimiting(rateLimitData)
	return annos, nil
}

// extractRateLimitData returns a set of annotations for rate limiting given the rate limit headers provided by HubSpot.
func extractRateLimitData(response *http.Response) (*v2.RateLimitDescription, error) {
	if response == nil {
		return nil, fmt.Errorf("hubspot-connector: passed nil response")
	}

	var (
		err       error
		remaining int64
	)

	remainingPayload := response.Header.Get("X-HubSpot-RateLimit-Remaining")
	if remainingPayload != "" {
		remaining, err = strconv.ParseInt(remainingPayload, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ratelimit-remaining: %w", err)
		}
	}

	var maxValue int64
	maxPayload := response.Header.Get("X-HubSpot-RateLimit-Max")
	if maxPayload != "" {
		maxValue, err = strconv.ParseInt(maxPayload, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ratelimit-max: %w", err)
		}
	}

	var resetAt *timestamppb.Timestamp
	intervalMsPayload := response.Header.Get("X-HubSpot-RateLimit-Interval-Milliseconds")
	if intervalMsPayload != "" {
		intervalMs, err := strconv.ParseInt(intervalMsPayload, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ratelimit-interval-milliseconds: %w", err)
		}

		resetAtSeconds := time.Now().Add(time.Duration(intervalMs) * time.Millisecond).Unix()
		resetAt = &timestamppb.Timestamp{Seconds: resetAtSeconds}
	}

	return &v2.RateLimitDescription{
		Limit:     maxValue,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}
