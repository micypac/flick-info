package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Define a Permission slice that will hold the permissions code for a single user.
type Permissions []string

// Helper method to check whether the Permission slice contains a specific permission code.
func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}

	return false
}

// PermissionModel type.
type PermissionModel struct {
	DB *sql.DB
}

// GetAllForUser() method returns all permission codes for a specific user in a Permissions slice.
func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	stmt := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, stmt, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}


// Add the permission codes for a specific user. 
// User variadic parameter for the codes to assign multiple permissions in a single call.
func (m PermissionModel) AddForUser(userID int64, codes ...string) error {
	stmt := `
		INSERT INTO users_permissions
		SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, stmt, userID, pq.Array(codes))
	return err
}
