package domain

import "testing"

func TestPURINV002ExactMatchHasZeroTolerance(t *testing.T) {
	expected := []InvoiceLine{{ReferenceID: "line-1", Quantity: 10, UnitPrice: 12500}}
	if err := ExactMatch(expected, expected); err != nil {
		t.Fatal(err)
	}
	if err := ExactMatch(expected, []InvoiceLine{{ReferenceID: "line-1", Quantity: 9.9999, UnitPrice: 12500}}); err == nil {
		t.Fatal("quantity variance accepted")
	}
	if err := ExactMatch(expected, []InvoiceLine{{ReferenceID: "line-1", Quantity: 10, UnitPrice: 12500.01}}); err == nil {
		t.Fatal("price variance accepted")
	}
}

func TestPURPAY002PartialPaymentAndOverpayment(t *testing.T) {
	invoice := Invoice{Status: Posted, Total: 100000, Outstanding: 100000}
	partial, err := invoice.ApplyPayment(40000)
	if err != nil || partial.Status != PartiallyPaid || partial.Outstanding != 60000 {
		t.Fatalf("partial=%#v err=%v", partial, err)
	}
	paid, err := partial.ApplyPayment(60000)
	if err != nil || paid.Status != Paid || paid.Outstanding != 0 {
		t.Fatalf("paid=%#v err=%v", paid, err)
	}
	if _, err = partial.ApplyPayment(60000.01); err == nil {
		t.Fatal("overpayment accepted")
	}
}

func TestACTFINPaymentVoidRestoresOutstanding(t *testing.T) {
	invoice := Invoice{Status: Paid, Total: 100000, Outstanding: 0}
	reopened, err := invoice.ReversePayment(100000)
	if err != nil || reopened.Status != Posted || reopened.Outstanding != 100000 {
		t.Fatalf("reopened=%#v err=%v", reopened, err)
	}
}

func TestFINPaymentValidation(t *testing.T) {
	if err := ValidatePayment(100, 100, "IDR", "IDR", "2026-09-22", "2026-09-22"); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]any{
		"zero":     {float64(0), float64(0), "IDR", "IDR", "2026-09-22", "2026-09-22"},
		"sum":      {float64(100), float64(99), "IDR", "IDR", "2026-09-22", "2026-09-22"},
		"currency": {float64(100), float64(100), "IDR", "USD", "2026-09-22", "2026-09-22"},
		"date":     {float64(100), float64(100), "IDR", "IDR", "2026-09-21", "2026-09-22"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePayment(args[0].(float64), args[1].(float64), args[2].(string), args[3].(string), args[4].(string), args[5].(string)); err == nil {
				t.Fatal("invalid payment accepted")
			}
		})
	}
}
