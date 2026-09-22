package domain

import "testing"

func TestCAPMDM001MasterRecordValidation(t *testing.T) {
	if _, err := NewRecord(Product, " sku-1 ", "Produk Satu"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRecord(Product, "", "Produk"); err == nil {
		t.Fatal("empty code accepted")
	}
	if _, err := NewRecord(Kind("OTHER"), "X", "Other"); err == nil {
		t.Fatal("unknown kind accepted")
	}
}
