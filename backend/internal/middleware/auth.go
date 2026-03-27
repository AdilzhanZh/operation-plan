package middleware

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

const userContextKey = "current_user"

type UserContext struct {
	ID         int64  `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	MiddleName string `json:"middle_name"`
	FullName   string `json:"full_name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Position   string `json:"position"`
	Role       string `json:"role"`
}

func ExtractBearerToken(header string) (string, error) {
	authHeader := strings.TrimSpace(header)
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || token == authHeader {
		return "", fmt.Errorf("invalid bearer token")
	}

	return token, nil
}

func AuthRequired(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := ExtractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": err.Error()})
			return
		}

		var user UserContext
		err = db.QueryRow(`
			SELECT u.id,
			       u.first_name,
			       u.last_name,
			       u.middle_name,
			       u.full_name,
			       u.username,
			       COALESCE(u.email, ''),
			       COALESCE(u.position, ''),
			       u.role
			FROM user_sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.token = $1
			  AND s.expires_at > NOW()
			ORDER BY s.id DESC
			LIMIT 1
		`, token).Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.MiddleName,
			&user.FullName,
			&user.Username,
			&user.Email,
			&user.Position,
			&user.Role,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.AbortWithStatusJSON(401, gin.H{"error": "invalid session"})
				return
			}
			c.AbortWithStatusJSON(500, gin.H{"error": "failed to resolve session"})
			return
		}

		c.Set(userContextKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *UserContext {
	value, exists := c.Get(userContextKey)
	if !exists {
		return nil
	}

	user, ok := value.(UserContext)
	if !ok {
		return nil
	}

	return &user
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		if !slices.Contains(roles, user.Role) {
			c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}
