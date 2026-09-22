package domain

import "testing"

func TestSALSO002And004StateMachine(t *testing.T) {
	so := NewSalesOrder("so1", "operator1")
	if e := so.Submit(); e != nil {
		t.Fatal(e)
	}
	if e := so.Approve("operator1", true); e == nil {
		t.Fatal("maker approved own SO")
	}
	if e := so.Approve("admin1", false); e == nil {
		t.Fatal("approval ignored stock validation")
	}
	if e := so.Approve("admin1", true); e != nil {
		t.Fatal(e)
	}
}
func TestRejectedSalesOrderIsTerminal(t *testing.T) {
	so := NewSalesOrder("so1", "operator1")
	_ = so.Submit()
	_ = so.Reject("admin1", "invalid")
	if e := so.Submit(); e == nil {
		t.Fatal("rejected SO resubmitted")
	}
}
