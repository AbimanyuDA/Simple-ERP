package domain

import "errors"

type AdjustmentStatus string

const (
	AdjustmentDraft     AdjustmentStatus = "DRAFT"
	AdjustmentSubmitted AdjustmentStatus = "SUBMITTED"
	AdjustmentApproved  AdjustmentStatus = "APPROVED"
	AdjustmentCompleted AdjustmentStatus = "COMPLETED"
)

type Adjustment struct {
	ID, CreatorID string
	Status        AdjustmentStatus
	Version       int
}

func NewAdjustment(id, creator string) *Adjustment {
	return &Adjustment{ID: id, CreatorID: creator, Status: AdjustmentDraft, Version: 1}
}
func (a *Adjustment) Submit() error {
	if a.Status != AdjustmentDraft {
		return errors.New("invalid transition")
	}
	a.Status = AdjustmentSubmitted
	a.Version++
	return nil
}
func (a *Adjustment) Approve(actor string) error {
	if a.Status != AdjustmentSubmitted || actor == a.CreatorID {
		return errors.New("invalid transition or maker-checker")
	}
	a.Status = AdjustmentApproved
	a.Version++
	return nil
}
func (a *Adjustment) Complete() error {
	if a.Status != AdjustmentApproved {
		return errors.New("invalid transition")
	}
	a.Status = AdjustmentCompleted
	a.Version++
	return nil
}
