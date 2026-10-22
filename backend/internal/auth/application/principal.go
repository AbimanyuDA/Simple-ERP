package application

import (
	"context"
	"errors"
)

type Principal struct{ UserID, Email, DisplayName, Role, CSRFHash string }
type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}
func PrincipalFrom(ctx context.Context) (Principal, error) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	if !ok {
		return Principal{}, errors.New("principal missing")
	}
	return p, nil
}
