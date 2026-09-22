package domain

import "testing"

func TestOrganizationSettingsDefaultsAndValidation(t *testing.T) {
	s := OrganizationSettings{Currency: "IDR", Timezone: "Asia/Jakarta", DefaultWarehouseID: "w1"}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Currency = "US"
	if err := s.Validate(); err == nil {
		t.Fatal("invalid currency accepted")
	}
}
