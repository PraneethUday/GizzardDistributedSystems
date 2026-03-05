package models

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserRequest represents the request body for creating a user
type UserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

// UserWithShard represents a user along with their shard information
type UserWithShard struct {
	User    User `json:"user"`
	ShardID int  `json:"shard_id"`
}
