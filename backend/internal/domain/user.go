package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser             Role = "user"
	RoleUserAdmin        Role = "user_admin"
	RoleViceAdmin        Role = "vice_admin"
	RoleAssociationAdmin Role = "association_admin"
	RoleSystemAdmin      Role = "system_admin"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleUserAdmin, RoleViceAdmin, RoleAssociationAdmin, RoleSystemAdmin:
		return true
	}
	return false
}

// DisplayName はロールの日本語表示名を返す
func (r Role) DisplayName() string {
	switch r {
	case RoleSystemAdmin:
		return "システム管理者"
	case RoleAssociationAdmin:
		return "自治会長"
	case RoleViceAdmin:
		return "副会長"
	case RoleUserAdmin:
		return "ユーザー管理者"
	case RoleUser:
		return "一般ユーザー"
	}
	return string(r)
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
