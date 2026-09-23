package application

import "context"

type Summary struct {
	PendingApprovals   int     `json:"pendingApprovals"`
	ActivePurchases    int     `json:"activePurchases"`
	ActiveSales        int     `json:"activeSales"`
	ZeroStockItems     int     `json:"zeroStockItems"`
	Payables           float64 `json:"payables"`
	Receivables        float64 `json:"receivables"`
	OverduePayables    float64 `json:"overduePayables"`
	OverdueReceivables float64 `json:"overdueReceivables"`
}

type ReportRow struct {
	DocumentNo  string  `json:"documentNo"`
	Party       string  `json:"party"`
	Status      string  `json:"status"`
	Amount      float64 `json:"amount"`
	Outstanding float64 `json:"outstanding,omitempty"`
	DueDate     string  `json:"dueDate,omitempty"`
	DaysOverdue int     `json:"daysOverdue,omitempty"`
}

type InventoryRow struct {
	Warehouse string  `json:"warehouse"`
	SKU       string  `json:"sku"`
	Product   string  `json:"product"`
	Quantity  float64 `json:"quantity"`
}

type AuditRow struct {
	ID            int64  `json:"id"`
	OccurredAt    string `json:"occurredAt"`
	Actor         string `json:"actor"`
	Action        string `json:"action"`
	TraceID       string `json:"traceId"`
	EntityType    string `json:"entityType"`
	EntityID      string `json:"entityId"`
	PreviousState string `json:"previousState,omitempty"`
	NewState      string `json:"newState,omitempty"`
	Reason        string `json:"reason,omitempty"`
	RequestID     string `json:"requestId"`
}

type Store interface {
	Summary(context.Context) (Summary, error)
	Purchases(context.Context) ([]ReportRow, error)
	Sales(context.Context) ([]ReportRow, error)
	Inventory(context.Context) ([]InventoryRow, error)
	Payables(context.Context) ([]ReportRow, error)
	Receivables(context.Context) ([]ReportRow, error)
	Audit(context.Context, string, string, string) ([]AuditRow, error)
}

type Service struct{ store Store }

func NewService(store Store) Service                                  { return Service{store: store} }
func (s Service) Summary(c context.Context) (Summary, error)          { return s.store.Summary(c) }
func (s Service) Purchases(c context.Context) ([]ReportRow, error)    { return s.store.Purchases(c) }
func (s Service) Sales(c context.Context) ([]ReportRow, error)        { return s.store.Sales(c) }
func (s Service) Inventory(c context.Context) ([]InventoryRow, error) { return s.store.Inventory(c) }
func (s Service) Payables(c context.Context) ([]ReportRow, error)     { return s.store.Payables(c) }
func (s Service) Receivables(c context.Context) ([]ReportRow, error)  { return s.store.Receivables(c) }
func (s Service) Audit(c context.Context, entity, actor, action string) ([]AuditRow, error) {
	return s.store.Audit(c, entity, actor, action)
}
