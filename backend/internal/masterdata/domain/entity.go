package domain

import (
	"errors"
	"strings"
)

type Kind string

const (
	Product   Kind = "PRODUCT"
	Supplier  Kind = "SUPPLIER"
	Customer  Kind = "CUSTOMER"
	Unit      Kind = "UNIT"
	Warehouse Kind = "WAREHOUSE"
)

type Record struct {
	ID      string `json:"id"`
	Kind    Kind   `json:"kind"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Version int    `json:"version"`
}

func NewRecord(k Kind, code, name string) (Record, error) {
	if k != Product && k != Supplier && k != Customer && k != Unit && k != Warehouse {
		return Record{}, errors.New("invalid kind")
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return Record{}, errors.New("code and name required")
	}
	return Record{Kind: k, Code: code, Name: name, Active: true, Version: 1}, nil
}
