package v1

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,max=128"`
}

type CodeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code"  validate:"required,len=6,numeric"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
