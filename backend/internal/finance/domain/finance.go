package domain

import (
	"errors"
	"math"
	"strconv"
	"time"
)

type InvoiceStatus string

const (
	MatchPending   InvoiceStatus = "MATCH_PENDING"
	MatchException InvoiceStatus = "MATCH_EXCEPTION"
	Posted         InvoiceStatus = "POSTED"
	PartiallyPaid  InvoiceStatus = "PARTIALLY_PAID"
	Paid           InvoiceStatus = "PAID"
	Void           InvoiceStatus = "VOID"
)

type InvoiceLine struct {
	ReferenceID string
	Quantity    float64
	UnitPrice   float64
}

type Invoice struct {
	Status      InvoiceStatus
	Total       float64
	Outstanding float64
}

func MoneyMinorUnits(amount float64) (int64, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 {
		return 0, errors.New("invalid monetary amount")
	}
	normalized, err := strconv.ParseFloat(strconv.FormatFloat(amount, 'f', 2, 64), 64)
	if err != nil {
		return 0, errors.New("invalid monetary amount")
	}
	ulp := math.Abs(math.Nextafter(normalized, math.Inf(1)) - normalized)
	if math.Abs(normalized-amount) > ulp {
		return 0, errors.New("monetary amount exceeds two decimal places")
	}
	scaled := normalized * 100
	if scaled > math.MaxInt64 {
		return 0, errors.New("monetary amount exceeds supported range")
	}
	return int64(math.Round(scaled)), nil
}

func ExactMatch(expected, actual []InvoiceLine) error {
	if len(expected) != len(actual) {
		return errors.New("invoice line count does not match")
	}
	indexed := make(map[string]InvoiceLine, len(expected))
	for _, line := range expected {
		indexed[line.ReferenceID] = line
	}
	for _, line := range actual {
		expectedLine, ok := indexed[line.ReferenceID]
		if !ok || expectedLine.Quantity != line.Quantity || expectedLine.UnitPrice != line.UnitPrice {
			return errors.New("zero-tolerance quantity or price mismatch")
		}
	}
	return nil
}

func (i Invoice) ApplyPayment(amount float64) (Invoice, error) {
	if i.Status != Posted && i.Status != PartiallyPaid {
		return i, errors.New("invoice cannot receive payment")
	}
	if amount <= 0 || amount > i.Outstanding {
		return i, errors.New("payment exceeds outstanding")
	}
	i.Outstanding -= amount
	if i.Outstanding == 0 {
		i.Status = Paid
	} else {
		i.Status = PartiallyPaid
	}
	return i, nil
}

func (i Invoice) ReversePayment(amount float64) (Invoice, error) {
	if i.Status != Paid && i.Status != PartiallyPaid {
		return i, errors.New("invoice has no posted payment")
	}
	if amount <= 0 || i.Outstanding+amount > i.Total {
		return i, errors.New("invalid reversal amount")
	}
	i.Outstanding += amount
	if i.Outstanding == i.Total {
		i.Status = Posted
	} else {
		i.Status = PartiallyPaid
	}
	return i, nil
}

func ValidatePayment(amount, allocationTotal float64, paymentCurrency, invoiceCurrency, paymentDate, invoiceDate string) error {
	amountMinor, amountErr := MoneyMinorUnits(amount)
	allocationMinor, allocationErr := MoneyMinorUnits(allocationTotal)
	if amountErr != nil || allocationErr != nil || amountMinor <= 0 || amountMinor != allocationMinor {
		return errors.New("payment amount and allocations must be positive and equal")
	}
	if paymentCurrency != invoiceCurrency {
		return errors.New("payment currency mismatch")
	}
	paymentAt, err := time.Parse(time.DateOnly, paymentDate)
	if err != nil {
		return errors.New("invalid payment date")
	}
	invoiceAt, err := time.Parse(time.DateOnly, invoiceDate)
	if err != nil || paymentAt.Before(invoiceAt) {
		return errors.New("payment date precedes invoice date")
	}
	return nil
}
