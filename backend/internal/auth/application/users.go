package application

import "context"

type UserView struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	Version     int    `json:"version"`
}
type UserAdminStore interface {
	ListUsers(context.Context) ([]UserView, error)
	SetUserStatus(context.Context, string, string, int, string, string) error
	RevokeSessions(context.Context, string, string, string) error
}
type UserAdminService struct{ store UserAdminStore }

func NewUserAdminService(s UserAdminStore) UserAdminService { return UserAdminService{s} }
func (s UserAdminService) List(ctx context.Context) ([]UserView, error) {
	return s.store.ListUsers(ctx)
}
func (s UserAdminService) SetStatus(ctx context.Context, id, status string, version int, actor, request string) error {
	return s.store.SetUserStatus(ctx, id, status, version, actor, request)
}
func (s UserAdminService) Revoke(ctx context.Context, id, actor, request string) error {
	return s.store.RevokeSessions(ctx, id, actor, request)
}
