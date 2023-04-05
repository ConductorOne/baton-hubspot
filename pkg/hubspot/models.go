package hubspot

type BaseResource struct {
	Id string `json:"id"`
}

type User struct {
	BaseResource
	Email  string `json:"email"`
	RoleId string `json:"roleId"`
	TeamId string `json:"primaryTeamId"`
}

type Team struct {
	BaseResource
	Name             string   `json:"name"`
	UserIds          []string `json:"userIds"`
	SecondaryUserIds []string `json:"secondaryUserIds"`
}

type Account struct {
	Id   int    `json:"portalId"`
	Type string `json:"accountType"`
}

type Role struct {
	BaseResource
	Name string `json:"name"`
	// TODO: reserach this with enterprise api key
	// Permissions []Permission `json:"permissions"`
}

// type Permission struct {
// 	BaseResource
// 	Name string `json:"name"`
// }

type Page struct {
	After string `json:"after"`
	Link  string `json:"link"`
}

type PaginationData struct {
	Next Page `json:"next"`
}
