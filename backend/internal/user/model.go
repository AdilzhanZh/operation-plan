package user

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleProrector Role = "prorector"
	RoleViewer    Role = "viewer"
)

type User struct {
	ID            int64  `json:"id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	MiddleName    string `json:"middle_name"`
	FullName      string `json:"full_name"`
	Username      string `json:"username"`
	PasswordPlain string `json:"password_plain"`
	Role          Role   `json:"role"`
	CreatedAt     string `json:"created_at"`
}
