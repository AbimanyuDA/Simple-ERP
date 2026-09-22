package domain

import (
	"errors"
	"strings"
)

type OrganizationSettings struct {
	OrganizationName   string `json:"organizationName"`
	Currency           string `json:"currency"`
	Timezone           string `json:"timezone"`
	DefaultWarehouseID string `json:"defaultWarehouseId"`
}

func (s OrganizationSettings) Validate() error {
	if len(s.Currency) != 3 || strings.TrimSpace(s.Timezone) == "" || strings.TrimSpace(s.DefaultWarehouseID) == "" {
		return errors.New("currency, timezone, and default warehouse are required")
	}
	return nil
}
