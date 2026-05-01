package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser             Role = "user"
	RoleAssociationAdmin Role = "association_admin"
	RoleSystemAdmin      Role = "system_admin"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAssociationAdmin, RoleSystemAdmin:
		return true
	}
	return false
}

type User struct {
	ID            uuid.UUID  `json:"id"`
	AssociationID *uuid.UUID `json:"association_id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"`
	Role          Role       `json:"role"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (u *User) IsSystemAdmin() bool {
	return u.Role == RoleSystemAdmin
}

func (u *User) CanAccessAssociation(associationID uuid.UUID) bool {
	if u.IsSystemAdmin() {
		return true
	}
	if u.AssociationID == nil {
		return false
	}
	return *u.AssociationID == associationID
}
