package domain

import "errors"

var (
	ErrInvalidTransition = errors.New("invalid purchase request transition")
	ErrMakerChecker      = errors.New("creator cannot approve or reject own transaction")
)

type PRStatus string

const (
	PRDraft     PRStatus = "DRAFT"
	PRSubmitted PRStatus = "SUBMITTED"
	PRApproved  PRStatus = "APPROVED"
	PRRejected  PRStatus = "REJECTED"
	PRConverted PRStatus = "CONVERTED_TO_PO"
	PRCompleted PRStatus = "COMPLETED"
	PRCancelled PRStatus = "CANCELLED"
)

type PurchaseRequest struct {
	ID, CreatorID, ReviewerID, Reason string
	Status                            PRStatus
	Version                           int
}

func NewPurchaseRequest(id, creator string) *PurchaseRequest {
	return &PurchaseRequest{ID: id, CreatorID: creator, Status: PRDraft, Version: 1}
}
func (p *PurchaseRequest) Submit() error {
	if p.Status != PRDraft {
		return ErrInvalidTransition
	}
	p.Status = PRSubmitted
	p.Version++
	return nil
}
func (p *PurchaseRequest) Approve(actor string) error {
	if p.Status != PRSubmitted {
		return ErrInvalidTransition
	}
	if actor == p.CreatorID {
		return ErrMakerChecker
	}
	p.Status = PRApproved
	p.ReviewerID = actor
	p.Version++
	return nil
}
func (p *PurchaseRequest) Reject(actor, reason string) error {
	if p.Status != PRSubmitted {
		return ErrInvalidTransition
	}
	if actor == p.CreatorID {
		return ErrMakerChecker
	}
	if reason == "" {
		return errors.New("reason required")
	}
	p.Status = PRRejected
	p.ReviewerID = actor
	p.Reason = reason
	p.Version++
	return nil
}
func (p PurchaseRequest) Copy(id, creator string) *PurchaseRequest {
	return NewPurchaseRequest(id, creator)
}
