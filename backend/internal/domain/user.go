package domain


type CreateUserDTO struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
}

type UserResponseDTO struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TenantID  string `json:"tenant_id"`
}

type User struct {
	ID        string `json:"id" db:"id"`
	FirstName string `json:"first_name" db:"name"` // Notar que en db es "name" en la tabla users
	LastName  string `json:"last_name" db:"-"`
	Email     string `json:"email" db:"email"`
	Role      string `json:"role" db:"role"`
	TenantID  string `json:"tenant_id" db:"tenant_id"`
}


