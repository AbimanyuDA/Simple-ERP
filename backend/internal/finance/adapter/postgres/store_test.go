package postgres

import (
	"testing"

	"github.com/simple-erp/backend/internal/finance/application"
)

func TestAllocationTotalUsesExactMinorUnits(t *testing.T) {
	total, err := allocationTotal([]application.AllocationInput{
		{InvoiceID: "invoice-1", Amount: 0.1},
		{InvoiceID: "invoice-2", Amount: 0.2},
	})
	if err != nil || total != 30 {
		t.Fatalf("total=%d err=%v", total, err)
	}
	if _, err = allocationTotal([]application.AllocationInput{{InvoiceID: "invoice-1", Amount: 10.001}}); err == nil {
		t.Fatal("allocation with unsupported scale accepted")
	}
}

func TestValidateReplacement(t *testing.T) {
	tests := []struct {
		name         string
		latestID     string
		latestStatus string
		requestedID  string
		hasPrior     bool
		want         bool
	}{
		{name: "first invoice without replacement", hasPrior: false, want: true},
		{name: "first invoice cannot claim replacement", requestedID: "old", hasPrior: false, want: false},
		{name: "active invoice blocks another", latestID: "active", latestStatus: "POSTED", hasPrior: true, want: false},
		{name: "void invoice requires link", latestID: "void-1", latestStatus: "VOID", hasPrior: true, want: false},
		{name: "older void invoice cannot be reused", latestID: "void-2", latestStatus: "VOID", requestedID: "void-1", hasPrior: true, want: false},
		{name: "latest void invoice can be replaced", latestID: "void-2", latestStatus: "VOID", requestedID: "void-2", hasPrior: true, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validateReplacement(test.latestID, test.latestStatus, test.requestedID, test.hasPrior); got != test.want {
				t.Fatalf("validateReplacement()=%t want=%t", got, test.want)
			}
		})
	}
}
