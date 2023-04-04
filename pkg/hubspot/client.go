package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

func setupPaginationQuery(query *url.Values, limit int, after string) {
	// add limit
	if limit != 0 {
		query.Add("limit", strconv.Itoa(limit))
	}

	// add page reference
	if after != "" {
		query.Add("after", after)
	}
}

// GetUsers returns all users for a single workspace.
func (c *Client) GetUsers(ctx context.Context, getUsersVars GetUsersVars) ([]User, string, *http.Response, error) {
	queryParamaters := url.Values{}
	setupPaginationQuery(&queryParamaters, getUsersVars.Limit, getUsersVars.After)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UsersBaseURL, nil)
	if err != nil {
		return nil, "", nil, err
	}

	req.URL.RawQuery = queryParamaters.Encode()
	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")

	rawResponse, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", nil, err
	}
	defer rawResponse.Body.Close()

	var userResponse UsersResponse
	if err := json.NewDecoder(rawResponse.Body).Decode(&userResponse); err != nil {
		return nil, "", nil, err
	}

	if (userResponse.Paging != PaginationData{}) {
		return userResponse.Results, userResponse.Paging.Next.After, rawResponse, nil
	}

	return userResponse.Results, "", rawResponse, nil
}

// GetTeams returns all teams for a single account.
func (c *Client) GetTeams(ctx context.Context) ([]Team, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, TeamsBaseURL, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")

	rawResponse, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer rawResponse.Body.Close()

	var teamResponse TeamsResponse
	if err := json.NewDecoder(rawResponse.Body).Decode(&teamResponse); err != nil {
		return nil, nil, err
	}

	return teamResponse.Results, rawResponse, nil
}

// GetAccount return informations about single account.
func (c *Client) GetAccount(ctx context.Context) (Account, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AccountBaseURL, nil)
	if err != nil {
		return Account{}, nil, err
	}

	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")

	rawResponse, err := c.httpClient.Do(req)
	if err != nil {
		return Account{}, nil, err
	}
	defer rawResponse.Body.Close()

	var accountResponse Account
	if err := json.NewDecoder(rawResponse.Body).Decode(&accountResponse); err != nil {
		return Account{}, nil, err
	}

	return accountResponse, rawResponse, nil
}

// GetRoles return all roles under a single account.
func (c *Client) GetRoles(ctx context.Context) ([]Role, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AccountBaseURL, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("authorization", fmt.Sprint("Bearer ", c.accessToken))
	req.Header.Add("accept", "application/json")

	rawResponse, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer rawResponse.Body.Close()

	var rolesResponse RolesResponse
	if err := json.NewDecoder(rawResponse.Body).Decode(&rolesResponse); err != nil {
		return nil, nil, err
	}

	return rolesResponse.Results, rawResponse, nil
}
