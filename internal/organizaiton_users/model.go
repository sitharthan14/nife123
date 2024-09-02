package oragnizationUsers

import "time"

type OrganizationUsers struct {
	Id        string    `json:"id"`
	UserId    string    `json:"user_id"`
	JoinedAt  time.Time `json:"joined_at"`
	UserRole  string    `json:"user_role"`
	UserEmail string    `json:"email"`
	UserName  string    `json:"user_name"`
	RoleId    int       `json:"role_id"`
	FirstName string    `json:"first_name"`
	LastName  string	`json:"last_name"`
}
