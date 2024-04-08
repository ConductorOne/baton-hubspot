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
	After string `json:"after"`
	Link  string `json:"link"`
}

type PaginationData struct {
	Next Page `json:"next"`
}
