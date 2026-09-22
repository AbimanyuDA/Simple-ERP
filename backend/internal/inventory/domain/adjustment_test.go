package domain

import "testing"

func TestINVADJ002And003MakerChecker(t *testing.T) {
	a := NewAdjustment("a1", "operator")
	if e := a.Submit(); e != nil {
		t.Fatal(e)
	}
	if e := a.Approve("operator"); e == nil {
		t.Fatal("maker approved adjustment")
	}
	if e := a.Approve("admin"); e != nil {
		t.Fatal(e)
	}
	if e := a.Complete(); e != nil {
		t.Fatal(e)
	}
}
