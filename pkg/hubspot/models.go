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

type Page struct {
	After string `json:"after"`
	Link  string `json:"link"`
}

type PaginationData struct {
	Next Page `json:"next"`
}
