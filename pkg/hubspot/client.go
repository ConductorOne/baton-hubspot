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
const AccountBaseURL = BaseURL + "account-info/v3"

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

func NewClient(accessToken string, httpClient *http.Client) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  httpClient,
	}
}

// returns query params with pagination options.
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
