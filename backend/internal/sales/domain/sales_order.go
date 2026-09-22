package domain

import "errors"

var (
	ErrInvalidTransition = errors.New("invalid sales order transition")
	ErrMakerChecker      = errors.New("creator cannot approve or reject own transaction")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type SOStatus string

const (
	SODraft          SOStatus = "DRAFT"
	SOSubmitted      SOStatus = "SUBMITTED"
	SOApproved       SOStatus = "APPROVED"
	SORejected       SOStatus = "REJECTED"
	SODelivering     SOStatus = "PARTIALLY_DELIVERED"
	SOFullyDelivered SOStatus = "FULLY_DELIVERED"
	SOCompleted      SOStatus = "COMPLETED"
	SOCancelled      SOStatus = "CANCELLED"
)

type SalesOrder struct {
	ID, CreatorID, ReviewerID, Reason string
	Status                            SOStatus
	Version                           int
}

func NewSalesOrder(id, creator string) *SalesOrder {
	return &SalesOrder{ID: id, CreatorID: creator, Status: SODraft, Version: 1}
}
func (s *SalesOrder) Submit() error {
	if s.Status != SODraft {
		return ErrInvalidTransition
	}
	s.Status = SOSubmitted
	s.Version++
	return nil
}
func (s *SalesOrder) Approve(actor string, stockAvailable bool) error {
	if s.Status != SOSubmitted {
		return ErrInvalidTransition
	}
	if actor == s.CreatorID {
		return ErrMakerChecker
	}
	if !stockAvailable {
		return ErrInsufficientStock
	}
	s.Status = SOApproved
	s.ReviewerID = actor
	s.Version++
	return nil
}
func (s *SalesOrder) Reject(actor, reason string) error {
	if s.Status != SOSubmitted {
		return ErrInvalidTransition
	}
	if actor == s.CreatorID {
		return ErrMakerChecker
	}
	if reason == "" {
		return errors.New("reason required")
	}
	s.Status = SORejected
	s.ReviewerID = actor
	s.Reason = reason
	s.Version++
	return nil
}
