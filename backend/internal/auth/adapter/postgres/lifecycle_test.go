package postgres

import (
	"testing"

	"github.com/simple-erp/backend/internal/auth/domain"
)

func TestConsumeStatusAllowed(t *testing.T) {
	tests := []struct {
		name   string
		kind   domain.ActionTokenKind
		status string
		want   bool
	}{
		{name: "invitation activates invited user", kind: domain.Invitation, status: "INVITED", want: true},
		{name: "invitation cannot reactivate deactivated user", kind: domain.Invitation, status: "DEACTIVATED", want: false},
		{name: "reset applies to active user", kind: domain.PasswordReset, status: "ACTIVE", want: true},
		{name: "reset cannot reactivate deactivated user", kind: domain.PasswordReset, status: "DEACTIVATED", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := consumeStatusAllowed(test.kind, test.status); got != test.want {
				t.Fatalf("consumeStatusAllowed()=%t want=%t", got, test.want)
			}
		})
	}
}
