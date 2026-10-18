package postgres

import (
	"context"
	"database/sql"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	// 🌟 Penjaga Nil-Safety Tipe Data Nullable Driver PGX v5
)

type sqlSupplierRepository struct {
	q *db.Queries
}

// NewSQLSupplierRepository menginisialisasi implementasi SupplierRepository berbasis pgxpool
func NewSQLSupplierRepository(pool *pgxpool.Pool) domain.SupplierRepository {
	return &sqlSupplierRepository{q: db.New(pool)}
}

func (r *sqlSupplierRepository) Create(ctx context.Context, supplier *domain.Supplier) error {
	// KONVERSI DEFENSIF AUDIT TRAIL: Cegah error foreign key constraint jika ID pengguna 0
	var createdByPg pgtype.Int8
	if supplier.CreatedBy != nil && *supplier.CreatedBy > 0 {
		createdByPg = pgtype.Int8{Int64: *supplier.CreatedBy, Valid: true}
	} else {
		createdByPg = pgtype.Int8{Valid: false}
	}

	var updatedByPg pgtype.Int8
	if supplier.UpdatedBy != nil && *supplier.UpdatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: *supplier.UpdatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	// KONVERSI FINANSIAL: Petakan tipe int32 opsional menjadi pgtype.Int4
	var bankIDPg pgtype.Int4
	if supplier.BankID != nil && *supplier.BankID > 0 {
		bankIDPg = pgtype.Int4{Int32: *supplier.BankID, Valid: true}
	} else {
		bankIDPg = pgtype.Int4{Valid: false}
	}

	// KONVERSI FINANSIAL: Petakan tipe string opsional menjadi pgtype.Text
	var accNumPg pgtype.Text
	if supplier.AccountNumber != "" {
		accNumPg = pgtype.Text{String: supplier.AccountNumber, Valid: true}
	} else {
		accNumPg = pgtype.Text{Valid: false}
	}

	var accNamePg pgtype.Text
	if supplier.AccountName != "" {
		accNamePg = pgtype.Text{String: supplier.AccountName, Valid: true}
	} else {
		accNamePg = pgtype.Text{Valid: false}
	}

	row, err := r.q.CreateSupplier(ctx, db.CreateSupplierParams{
		Name:          supplier.Name,
		PicName:       supplier.PicName,
		Phone:         supplier.Phone,
		Email:         supplier.Email,
		Address:       supplier.Address,
		VillageID:     supplier.VillageID,
		BankID:        bankIDPg,
		AccountNumber: accNumPg,
		AccountName:   accNamePg,
		CreatedBy:     createdByPg,
		UpdatedBy:     updatedByPg,
	})
	if err != nil {
		return err
	}

	// Ikat kembali properti hasil mutasi fisik database ke objek penampung domain
	supplier.ID = row.ID
	supplier.IsActive = row.IsActive
	supplier.CreatedAt = r.interfaceToTime(row.CreatedAt)
	return nil
}

func (r *sqlSupplierRepository) GetSuppliersCursor(ctx context.Context, search, province string, cursorID int64, limit int, direction string) ([]domain.Supplier, int64, error) {
	// Inisialisasi make array kosong murni [] agar respons JSON tidak melahirkan null jika database kosong
	suppliers := make([]domain.Supplier, 0)

	// 1. Ambil hitungan total data riil database untuk kontrol penanda halaman frontend
	totalCount, err := r.q.CountSuppliers(ctx, db.CountSuppliersParams{
		SearchQuery:    search,
		ProvinceFilter: province,
	})
	if err != nil {
		return nil, 0, err
	}

	// 2. Pemisahan arah jalurnya kursor navigasi (Next arah maju atau Prev arah mundur)
	if direction == "prev" {
		rows, err := r.q.GetSuppliersCursorPrev(ctx, db.GetSuppliersCursorPrevParams{
			SearchQuery:    search,
			ProvinceFilter: province,
			CursorID:       cursorID,
			ReqLimit:       int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}

		reverseRows(rows)
		for _, row := range rows {
			suppliers = append(suppliers, r.mapRowToDomain(row))
		}
	} else {
		rows, err := r.q.GetSuppliersCursorNext(ctx, db.GetSuppliersCursorNextParams{
			SearchQuery:    search,
			ProvinceFilter: province,
			CursorID:       cursorID,
			ReqLimit:       int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}

		for _, row := range rows {
			suppliers = append(suppliers, r.mapRowToDomain(row))
		}
	}

	return suppliers, totalCount, nil
}

func (r *sqlSupplierRepository) GetByID(ctx context.Context, id int64) (*domain.Supplier, error) {
	row, err := r.q.GetSupplierByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 🌟 PERBAIKAN MUTLAK: Gunakan pemetaan manual kaku yang aman dari pembacaan database
	return &domain.Supplier{
		ID:           row.ID,
		Name:         row.Name,
		PicName:      row.PicName,
		Phone:        row.Phone,
		Email:        row.Email,
		Address:      row.Address,
		VillageID:    row.VillageID,
		VillageName:  r.interfaceToString(row.VillageName),
		PostalCode:   r.interfaceToString(row.PostalCode),
		DistrictID:   r.interfaceToString(row.DistrictID),
		DistrictName: r.interfaceToString(row.DistrictName),
		RegencyID:    r.interfaceToString(row.RegencyID),
		RegencyName:  r.interfaceToString(row.RegencyName),
		ProvinceID:   r.interfaceToString(row.ProvinceID),
		ProvinceName: r.interfaceToString(row.ProvinceName),

		// 🌟 AMANKAN FINANSIAL: Ambil properti bank dengan interfaceToString pembantu dinamis
		BankID:        r.interfaceToInt32Ptr(row.BankID),
		BankKode:      r.interfaceToString(row.BankKode),
		BankName:      r.interfaceToString(row.BankName),
		AccountNumber: r.interfaceToString(row.AccountNumber),
		AccountName:   r.interfaceToString(row.AccountName),

		IsActive:  row.IsActive,
		CreatedAt: r.interfaceToTime(row.CreatedAt),
	}, nil
}

func (r *sqlSupplierRepository) Update(ctx context.Context, supplier *domain.Supplier) error {
	var updatedByPg pgtype.Int8
	if supplier.UpdatedBy != nil && *supplier.UpdatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: *supplier.UpdatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	var bankIDPg pgtype.Int4
	if supplier.BankID != nil && *supplier.BankID > 0 {
		bankIDPg = pgtype.Int4{Int32: *supplier.BankID, Valid: true}
	} else {
		bankIDPg = pgtype.Int4{Valid: false}
	}

	var accNumPg pgtype.Text
	if supplier.AccountNumber != "" {
		accNumPg = pgtype.Text{String: supplier.AccountNumber, Valid: true}
	} else {
		accNumPg = pgtype.Text{Valid: false}
	}

	var accNamePg pgtype.Text
	if supplier.AccountName != "" {
		accNamePg = pgtype.Text{String: supplier.AccountName, Valid: true}
	} else {
		accNamePg = pgtype.Text{Valid: false}
	}

	return r.q.UpdateSupplier(ctx, db.UpdateSupplierParams{
		ID:            supplier.ID,
		Name:          supplier.Name,
		PicName:       supplier.PicName,
		Phone:         supplier.Phone,
		Email:         supplier.Email,
		Address:       supplier.Address,
		VillageID:     supplier.VillageID,
		BankID:        bankIDPg,
		AccountNumber: accNumPg,
		AccountName:   accNamePg,
		IsActive:      supplier.IsActive,
		UpdatedBy:     updatedByPg,
	})
}

func (r *sqlSupplierRepository) Delete(ctx context.Context, id, deletedBy int64) error {
	deletedByPg := pgtype.Int8{Int64: deletedBy, Valid: true}

	return r.q.DeleteSupplier(ctx, db.DeleteSupplierParams{
		ID:        id,
		DeletedBy: deletedByPg,
	})
}

// =========================================================================
// 🔒 UTILITY REFACTORING: REFLECTION MAPPER UNTUK NEXT & PREV ROW DATA
// =========================================================================

func (r *sqlSupplierRepository) mapRowToDomain(row any) domain.Supplier {
	// 🌟 FIX UTAMA: Gunakan runtime reflection map. sqlc memproduksi struct NextRow dan PrevRow berbeda.
	// Jika row.BankName melahirkan error undefined, helper ini mengisolasi penamaan secara dinamis.

	s := domain.Supplier{}

	switch v := row.(type) {
	case db.GetSuppliersCursorNextRow:
		s.ID = v.ID
		s.Name = v.Name
		s.PicName = v.PicName
		s.Phone = v.Phone
		s.Email = v.Email
		s.Address = v.Address
		s.VillageID = v.VillageID
		s.VillageName = r.interfaceToString(v.VillageName)
		s.PostalCode = r.interfaceToString(v.PostalCode)
		s.DistrictID = r.interfaceToString(v.DistrictID)
		s.DistrictName = r.interfaceToString(v.DistrictName)
		s.RegencyID = r.interfaceToString(v.RegencyID)
		s.RegencyName = r.interfaceToString(v.RegencyName)
		s.ProvinceID = r.interfaceToString(v.ProvinceID)
		s.ProvinceName = r.interfaceToString(v.ProvinceName)

		// Ekstraksi Rekening Finansial Berbasis Interface Adaptif
		s.BankID = r.interfaceToInt32Ptr(v.BankID)
		s.BankKode = r.interfaceToString(v.BankKode)
		s.BankName = r.interfaceToString(v.BankName)
		s.AccountNumber = r.interfaceToString(v.AccountNumber)
		s.AccountName = r.interfaceToString(v.AccountName)

		s.IsActive = v.IsActive
		s.CreatedAt = r.interfaceToTime(v.CreatedAt)

	case db.GetSuppliersCursorPrevRow:
		s.ID = v.ID
		s.Name = v.Name
		s.PicName = v.PicName
		s.Phone = v.Phone
		s.Email = v.Email
		s.Address = v.Address
		s.VillageID = v.VillageID
		s.VillageName = r.interfaceToString(v.VillageName)
		s.PostalCode = r.interfaceToString(v.PostalCode)
		s.DistrictID = r.interfaceToString(v.DistrictID)
		s.DistrictName = r.interfaceToString(v.DistrictName)
		s.RegencyID = r.interfaceToString(v.RegencyID)
		s.RegencyName = r.interfaceToString(v.RegencyName)
		s.ProvinceID = r.interfaceToString(v.ProvinceID)
		s.ProvinceName = r.interfaceToString(v.ProvinceName)

		// Ekstraksi Rekening Finansial Berbasis Interface Adaptif
		s.BankID = r.interfaceToInt32Ptr(v.BankID)
		s.BankKode = r.interfaceToString(v.BankKode)
		s.BankName = r.interfaceToString(v.BankName)
		s.AccountNumber = r.interfaceToString(v.AccountNumber)
		s.AccountName = r.interfaceToString(v.AccountName)

		s.IsActive = v.IsActive
		s.CreatedAt = r.interfaceToTime(v.CreatedAt)
	}

	return s
}

// =========================================================================
// 🔒 UTILITAS PENYELESAI EROR KETIDAKSESUAIAN DATA (NIL-SAFETY HELPERS)
// =========================================================================

func (r *sqlSupplierRepository) interfaceToString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case *string:
		if v != nil {
			return *v
		}
	case sql.NullString:
		if v.Valid {
			return v.String
		}
	case pgtype.Text:
		if v.Valid {
			return v.String
		}
	}
	return ""
}

func (r *sqlSupplierRepository) interfaceToTime(val any) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		return v
	case *time.Time:
		if v != nil {
			return *v
		}
	case pgtype.Timestamp:
		if v.Valid {
			return v.Time
		}
	case pgtype.Timestamptz:
		if v.Valid {
			return v.Time
		}
	}
	return time.Time{}
}

func (r *sqlSupplierRepository) interfaceToInt32Ptr(val any) *int32 {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case int32:
		return &v
	case *int32:
		return v
	case pgtype.Int4:
		if v.Valid {
			return &v.Int32
		}
	}
	return nil
}
