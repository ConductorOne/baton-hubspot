package hubspot

type BaseResource struct {
	Id string `json:"id"`
}

type User struct {
	BaseResource
	Email            string   `json:"email"`
	RoleIDs          []string `json:"roleIds"`
	TeamId           string   `json:"primaryTeamId"`
	SecondaryTeamIDs []string `json:"secondaryTeamIds"`
	SuperAdmin       bool     `json:"superAdmin"`
}

type Team struct {
	BaseResource
	Name             string   `json:"name"`
	UserIDs          []string `json:"userIds"`
	SecondaryUserIDs []string `json:"secondaryUserIds"`
}

type UserObject struct {
	BaseResource
	Properties UserObjectProperties `json:"properties,omitempty"`
}

type UserObjectProperties struct {
	UserId      string `json:"hs_internal_user_id,omitempty"`
	Deactivated string `json:"hs_deactivated,omitempty"`
}

type Account struct {
	Id   int    `json:"portalId"`
	Type string `json:"accountType"`
}

type Role struct {
	BaseResource
	Name string `json:"name"`
}

func NewRole(id, name string) *Role {
	return &Role{
		BaseResource: BaseResource{
			Id: id,
		},
		Name: name,
	}
}

type Page struct {
	After string `json:"after,omitempty"`
	Link  string `json:"link,omitempty"`
}

type PaginationData struct {
	Next Page `json:"next"`
}
