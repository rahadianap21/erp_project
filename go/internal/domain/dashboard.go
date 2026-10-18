package domain

import "context"

// DashboardStats merangkum angka KPI dashboard.
type DashboardStats struct {
	TotalPOValue   float64 `json:"total_po_value"`
	TotalPO        int64   `json:"total_po"`
	POPending      int64   `json:"po_pending"`
	POApproved     int64   `json:"po_approved"`
	POReceived     int64   `json:"po_received"`
	TotalGRN       int64   `json:"total_grn"`
	TotalProducts  int64   `json:"total_products"`
	TotalSuppliers int64   `json:"total_suppliers"`
	LowStockCount  int64   `json:"low_stock_count"`
}

// DashboardLowStockItem adalah produk aktif yang menyentuh ambang reorder.
type DashboardLowStockItem struct {
	ID              int64  `json:"id"`
	SKU             string `json:"sku"`
	Name            string `json:"name"`
	CurrentStock    int32  `json:"current_stock"`
	MinStockLevel   int32  `json:"min_stock_level"`
	SellingUnitKode string `json:"selling_unit"`
}

// DashboardRecentPO adalah baris dokumen PO terbaru untuk tabel dashboard.
type DashboardRecentPO struct {
	ID           int64   `json:"id"`
	PONumber     string  `json:"po_number"`
	SupplierName string  `json:"supplier_name"`
	TotalAmount  float64 `json:"total_amount"`
	Status       string  `json:"status"`
	PODate       string  `json:"po_date"`
}

// DashboardRecentGRN adalah baris penerimaan barang terbaru untuk feed dashboard.
type DashboardRecentGRN struct {
	ID           int64  `json:"id"`
	GRNNumber    string `json:"grn_number"`
	PONumber     string `json:"po_number"`
	TotalQty     int32  `json:"total_qty"`
	ReceivedDate string `json:"received_date"`
}

// DashboardOverview membungkus seluruh data yang dibutuhkan halaman dashboard.
type DashboardOverview struct {
	Stats      DashboardStats          `json:"stats"`
	LowStock   []DashboardLowStockItem `json:"low_stock"`
	RecentPOs  []DashboardRecentPO     `json:"recent_pos"`
	RecentGRNs []DashboardRecentGRN    `json:"recent_grns"`
}

type DashboardRepository interface {
	GetStats(ctx context.Context) (DashboardStats, error)
	GetLowStock(ctx context.Context, limit int) ([]DashboardLowStockItem, error)
	GetRecentPOs(ctx context.Context, limit int) ([]DashboardRecentPO, error)
	GetRecentGRNs(ctx context.Context, limit int) ([]DashboardRecentGRN, error)
}

type DashboardUsecase interface {
	GetOverview(ctx context.Context) (*DashboardOverview, error)
}
