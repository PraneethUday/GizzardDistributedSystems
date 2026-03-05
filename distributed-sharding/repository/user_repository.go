package repository

import (
	"database/sql"
	"fmt"

	"distributed-sharding/models"
)

// InsertUser inserts a user into the specified database shard
func InsertUser(db *sql.DB, user models.User) error {
	query := `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`
	_, err := db.Exec(query, user.ID, user.Name, user.Email)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

// GetUser retrieves a user from the specified database shard
func GetUser(db *sql.DB, id int) (*models.User, error) {
	query := `SELECT id, name, email FROM users WHERE id = ?`
	row := db.QueryRow(query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Name, &user.Email)
	if err == sql.ErrNoRows {
		return nil, nil // User not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetAllUsers retrieves all users from the specified database shard
func GetAllUsers(db *sql.DB) ([]models.User, error) {
	query := `SELECT id, name, email FROM users`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}

// UpdateUser updates a user in the specified database shard
func UpdateUser(db *sql.DB, user models.User) error {
	query := `UPDATE users SET name = ?, email = ? WHERE id = ?`
	result, err := db.Exec(query, user.Name, user.Email, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// DeleteUser deletes a user from the specified database shard
func DeleteUser(db *sql.DB, id int) error {
	query := `DELETE FROM users WHERE id = ?`
	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
