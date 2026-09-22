package domain

import "testing"

func TestSALDLV002NegativeStockIsProhibited(t *testing.T) {
	b := Balance{Quantity: 10}
	if e := b.Apply(-11); e == nil {
		t.Fatal("negative stock accepted")
	}
	if b.Quantity != 10 {
		t.Fatal("failed movement changed balance")
	}
}
func TestINVADJ004MovementIsSourceLinkedAndReversible(t *testing.T) {
	m, e := NewMovement("m1", "w1", "p1", -3, "STOCK_ADJUSTMENT", "adj1")
	if e != nil {
		t.Fatal(e)
	}
	r := m.Reversal("m2", "adj-reversal1")
	if r.Quantity != 3 || r.ReversalOf != "m1" || r.SourceID != "adj-reversal1" {
		t.Fatal("invalid reversal")
	}
}
