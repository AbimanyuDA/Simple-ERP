package domain

import (
	"strings"
	"time"
)

type UserStatus string

const (
	UserInvited     UserStatus = "INVITED"
	UserActive      UserStatus = "ACTIVE"
	UserDeactivated UserStatus = "DEACTIVATED"
)

type User struct {
	ID           string
	Email        string
	DisplayName  string
	Role         string
	Status       UserStatus
	PasswordHash string
	InvitedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func InviteUser(id, email, role, actorID string, at time.Time) (*User, error) {
	role = strings.ToUpper(strings.TrimSpace(role))
	if !ValidRole(role) {
		return nil, ErrInvalidRole
	}
	return &User{ID: id, Email: strings.ToLower(strings.TrimSpace(email)), Role: role, Status: UserInvited, InvitedBy: actorID, CreatedAt: at, UpdatedAt: at}, nil
}

func (u *User) Activate(passwordHash string, at time.Time) error {
	if u.Status != UserInvited || passwordHash == "" {
		return ErrInvalidState
	}
	u.PasswordHash, u.Status, u.UpdatedAt = passwordHash, UserActive, at
	return nil
}

func (u *User) Deactivate(at time.Time) error {
	if u.Status != UserActive {
		return ErrInvalidState
	}
	u.Status, u.UpdatedAt = UserDeactivated, at
	return nil
}
