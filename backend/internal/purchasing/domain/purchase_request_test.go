package domain

import "testing"

func TestPURPR002And003StateMachine(t *testing.T) {
	pr := NewPurchaseRequest("pr1", "operator1")
	if e := pr.Submit(); e != nil {
		t.Fatal(e)
	}
	if e := pr.Approve("operator1"); e == nil {
		t.Fatal("maker approved own PR")
	}
	if e := pr.Approve("admin1"); e != nil {
		t.Fatal(e)
	}
	if e := pr.Reject("admin1", "late"); e == nil {
		t.Fatal("approved PR changed arbitrarily")
	}
}
func TestRejectedPurchaseRequestIsTerminalAndCopyCreatesDraft(t *testing.T) {
	pr := NewPurchaseRequest("pr1", "operator1")
	_ = pr.Submit()
	if e := pr.Reject("admin1", "not needed"); e != nil {
		t.Fatal(e)
	}
	if e := pr.Submit(); e == nil {
		t.Fatal("rejected PR resubmitted")
	}
	copy := pr.Copy("pr2", "operator1")
	if copy.Status != PRDraft || copy.ID == pr.ID {
		t.Fatal("copy is not independent draft")
	}
}
