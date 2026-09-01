package domain

type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"` //nolint:gosec
	Username string `json:"username"`
}

type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"` //nolint:gosec
	Username string `json:"username" validate:"required,min=3"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"` //nolint:gosec
}

type UserResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type LoginResponse struct {
	Token string        `json:"token"`
	User  *UserResponse `json:"user"`
}
