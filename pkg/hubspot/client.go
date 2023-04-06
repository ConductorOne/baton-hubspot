package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const BaseURL = "https://api.hubapi.com/"
const UsersBaseURL = BaseURL + "settings/v3/users"
const TeamsBaseURL = BaseURL + "settings/v3/users/teams"
const AccountBaseURL = BaseURL + "account-info/v3/details"

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
func (c *Client) GetUsers(ctx context.Context, getUsersVars GetUsersVars) ([]User, string, error) {
	queryParams := setupPaginationQuery(url.Values{}, getUsersVars.Limit, getUsersVars.After)
	var userResponse UsersResponse

	err := c.doRequest(
		ctx,
		UsersBaseURL,
		&userResponse,
		queryParams,
	)

	if err != nil {
		return nil, "", err
	}

	if (userResponse.Paging != PaginationData{}) {
		return userResponse.Results, userResponse.Paging.Next.After, nil
	}

	return userResponse.Results, "", nil
}

// GetTeams returns all teams for a single account.
func (c *Client) GetTeams(ctx context.Context) ([]Team, error) {
	var teamResponse TeamsResponse
	err := c.doRequest(
		ctx,
		TeamsBaseURL,
		&teamResponse,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return teamResponse.Results, nil
}

// GetAccount returns information about single account.
func (c *Client) GetAccount(ctx context.Context) (Account, error) {
	var accountResponse Account
	err := c.doRequest(
		ctx,
		AccountBaseURL,
		&accountResponse,
		nil,
	)

	if err != nil {
		return Account{}, err
	}

	return accountResponse, nil
}

// GetUser returns information about a single user.
func (c *Client) GetUser(ctx context.Context, userId string) (User, error) {
	url := fmt.Sprint(UsersBaseURL, "/", userId)

	var userResponse User
	err := c.doRequest(
		ctx,
		url,
		&userResponse,
		nil,
	)
	if err != nil {
		return User{}, err
	}

	return userResponse, nil
}

// GetRoles returns all roles under a single account.
func (c *Client) GetRoles(ctx context.Context) ([]Role, error) {
	var rolesResponse RolesResponse
	err := c.doRequest(ctx, AccountBaseURL, &rolesResponse, nil)

	if err != nil {
		return nil, err
	}

	return rolesResponse.Results, nil
}

func (c *Client) doRequest(ctx context.Context, url string, resourceResponse interface{}, queryParams url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}

	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")

	rawResponse, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer rawResponse.Body.Close()

	if rawResponse.StatusCode >= 300 {
		return status.Error(codes.Code(rawResponse.StatusCode), "Request failed")
	}

	if err := json.NewDecoder(rawResponse.Body).Decode(&resourceResponse); err != nil {
		return err
	}

	return nil
}
