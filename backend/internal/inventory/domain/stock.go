package domain

import "errors"

var ErrNegativeStock = errors.New("stock cannot be negative")

type Balance struct{ Quantity int64 }

func (b *Balance) Apply(delta int64) error {
	if b.Quantity+delta < 0 {
		return ErrNegativeStock
	}
	b.Quantity += delta
	return nil
}

type Movement struct {
	ID, WarehouseID, ProductID, Kind, SourceID, ReversalOf string
	Quantity                                               int64
}

func NewMovement(id, warehouse, product string, qty int64, kind, source string) (Movement, error) {
	if id == "" || warehouse == "" || product == "" || source == "" || qty == 0 {
		return Movement{}, errors.New("movement requires identity, source, and non-zero quantity")
	}
	return Movement{ID: id, WarehouseID: warehouse, ProductID: product, Quantity: qty, Kind: kind, SourceID: source}, nil
}
func (m Movement) Reversal(id, source string) Movement {
	return Movement{ID: id, WarehouseID: m.WarehouseID, ProductID: m.ProductID, Quantity: -m.Quantity, Kind: m.Kind + "_REVERSAL", SourceID: source, ReversalOf: m.ID}
}
