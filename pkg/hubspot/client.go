package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const BaseURL = "https://api.hubapi.com/"
const UsersBaseURL = BaseURL + "settings/v3/users"
const UserBaseURL = BaseURL + "settings/v3/users/%s"
const TeamsBaseURL = BaseURL + "settings/v3/users/teams"
const RolesBaseURL = BaseURL + "settings/v3/users/roles"
const AccountBaseURL = BaseURL + "account-info/v3/details"
const SearchUserObjectURL = BaseURL + "crm/v3/objects/users/search"
const EqualOperator = "EQ"
const HSInternalUserId = "hs_internal_user_id"

type Client struct {
	httpClient  *http.Client
	accessToken string
}

type UsersResponse struct {
	Results []User         `json:"results"`
	Paging  PaginationData `json:"paging"`
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

func NewClient(accessToken string, httpClient *http.Client) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  httpClient,
	}
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
		UsersBaseURL,
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
		TeamsBaseURL,
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
		AccountBaseURL,
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
		fmt.Sprintf(UserBaseURL, userId),
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
	annos, err := c.get(ctx, RolesBaseURL, &rolesResponse, nil)
	if err != nil {
		return nil, nil, err
	}

	return rolesResponse.Results, annos, nil
}

type UpdateUserPayload struct {
	RoleId           string   `json:"roleId,omitempty"`
	PrimaryTeamId    string   `json:"primaryTeamId,omitempty"`
	SecondaryTeamIDs []string `json:"secondaryTeamIds,omitempty"`
}

// UpdateUser updates information about provided user.
func (c *Client) UpdateUser(ctx context.Context, userId string, payload *UpdateUserPayload) (annotations.Annotations, error) {
	annos, err := c.put(
		ctx,
		fmt.Sprintf(UserBaseURL, userId),
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
		SearchUserObjectURL,
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

func (c *Client) get(ctx context.Context, url string, resourceResponse interface{}, queryParams url.Values) (annotations.Annotations, error) {
	return c.doRequest(ctx, url, http.MethodGet, nil, resourceResponse, queryParams)
}

func (c *Client) put(ctx context.Context, url string, data interface{}, resourceResponse interface{}) (annotations.Annotations, error) {
	return c.doRequest(ctx, url, http.MethodPut, data, resourceResponse, nil)
}

func (c *Client) post(ctx context.Context, url string, data interface{}, resourceResponse interface{}) (annotations.Annotations, error) {
	return c.doRequest(ctx, url, http.MethodPost, data, resourceResponse, nil)
}

func (c *Client) doRequest(
	ctx context.Context,
	urlAddress string,
	method string,
	data interface{},
	resourceResponse interface{},
	queryParams url.Values,
) (annotations.Annotations, error) {
	var body io.Reader

	if data != nil {
		jsonBody, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		body = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlAddress, body)
	if err != nil {
		return nil, err
	}

	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}

	req.Header.Add("Authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	rawResponse, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer rawResponse.Body.Close()

	if rawResponse.StatusCode >= 300 {
		return nil, status.Error(codes.Code(rawResponse.StatusCode), "Request failed") //nolint:gosec // safe conversion: HTTP status code is always in range 0-599
	}

	if err := json.NewDecoder(rawResponse.Body).Decode(&resourceResponse); err != nil {
		return nil, err
	}

	rateLimitData, err := extractRateLimitData(rawResponse)
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
