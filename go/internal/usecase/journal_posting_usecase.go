package usecase

import (
	"context"
	"fmt"
	"time"

	"go_backend/internal/domain"
)

// PlanSaleJournal membangun rencana jurnal penjualan:
//
//	Debit  akun kanal pembayaran (kas/bank/e-wallet) per pembayaran,
//	       dikurangi kembalian tunai yang diserahkan ke pembeli
//	Debit  Piutang Usaha (AR) untuk sisa yang belum dibayar
//	Debit  HPP (COGS) sebesar total harga pokok
//	Kredit Pendapatan Penjualan sebesar DPP
//	Kredit PPN Keluaran (bila ada)
//	Kredit Persediaan sebesar total harga pokok
//
// paymentAccounts memetakan payment_method_id ke akun CoA. Rencana ini belum
// ditulis; repository penjualan yang menulisnya di dalam transaksi nota.
func (u *journalUsecase) PlanSaleJournal(ctx context.Context, sale *domain.Sale, paymentAccounts map[int64]int64, actorID int64) (*domain.SaleJournalPlan, error) {
	if sale == nil {
		return nil, domain.NewBusinessRuleError("jurnal penjualan ditolak: nota tidak valid")
	}

	var mapping map[string]int64
	var err error
	if u.settings != nil {
		mapping, err = u.settings.ListMapping(ctx)
		if err != nil {
			return nil, err
		}
	}

	revenueID, err := requireMapping(mapping, "SALES_REVENUE")
	if err != nil {
		return nil, err
	}
	var vatOutputID int64
	if sale.TaxAmount > 0 {
		if vatOutputID, err = requireMapping(mapping, "VAT_OUTPUT"); err != nil {
			return nil, err
		}
	}
	var inventoryID, cogsID int64
	if sale.CostTotal > 0 {
		if inventoryID, err = requireMapping(mapping, "INVENTORY"); err != nil {
			return nil, err
		}
		if cogsID, err = requireMapping(mapping, "COGS"); err != nil {
			return nil, err
		}
	}
	var arID int64
	unpaid := sale.UnpaidAmount()
	if unpaid > 0 {
		if arID, err = requireMapping(mapping, "AR"); err != nil {
			return nil, err
		}
	}

	entryDate := sale.SaleDate
	if entryDate.IsZero() {
		entryDate = time.Now()
	}

	lines := make([]domain.JournalLine, 0, len(sale.Payments)+4)
	// Debit kanal pembayaran. Kembalian uang tunai tidak mengalir masuk kas,
	// jadi dikurangi dari nominal pembayaran (kasir menyerahkan kembalian).
	changeLeft := sale.ChangeAmount
	for _, p := range sale.Payments {
		if p.Amount <= 0 {
			continue
		}
		acc, ok := paymentAccounts[p.PaymentMethodID]
		if !ok || acc <= 0 {
			return nil, fmt.Errorf("%w: metode pembayaran id %d belum memiliki akun", domain.ErrAutoPostingSettings, p.PaymentMethodID)
		}
		debit := p.Amount
		if changeLeft > 0 {
			applied := changeLeft
			if applied > debit {
				applied = debit
			}
			debit = round2(debit - applied)
			changeLeft = round2(changeLeft - applied)
		}
		if debit <= 0 {
			continue
		}
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   acc,
			Description: fmt.Sprintf("Penerimaan pembayaran - %s", paymentMethodLabel(p)),
			Debit:       debit,
		})
	}
	if unpaid > 0 {
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   arID,
			Description: "Piutang penjualan",
			Debit:       unpaid,
		})
	}
	if sale.CostTotal > 0 {
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   cogsID,
			Description: "Harga pokok penjualan",
			Debit:       sale.CostTotal,
		})
	}
	lines = append(lines, domain.JournalLine{
		LineNo:      len(lines) + 1,
		AccountID:   revenueID,
		Description: "Pendapatan penjualan",
		Credit:      sale.DppAmount,
	})
	if sale.TaxAmount > 0 {
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   vatOutputID,
			Description: "PPN keluaran",
			Credit:      sale.TaxAmount,
		})
	}
	if sale.CostTotal > 0 {
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   inventoryID,
			Description: "Pengurangan persediaan",
			Credit:      sale.CostTotal,
		})
	}

	return u.finalizeSaleJournalPlan(ctx, sale, entryDate, lines, nil, actorID)
}

// PlanSaleReversal membangun jurnal pembalik dari baris jurnal asli (debit dan
// kredit ditukar), lalu menandai jurnal asli sebagai sumber pembalikan.
func (u *journalUsecase) PlanSaleReversal(ctx context.Context, sale *domain.Sale, reason string, actorID int64) (*domain.SaleJournalPlan, error) {
	if sale == nil {
		return nil, domain.NewBusinessRuleError("pembatalan penjualan ditolak: nota tidak valid")
	}
	if sale.JournalEntryID == nil || *sale.JournalEntryID <= 0 {
		return nil, nil
	}
	orig, err := u.repo.GetByID(ctx, *sale.JournalEntryID)
	if err != nil {
		return nil, err
	}
	if orig == nil {
		return nil, domain.ErrJournalNotFound
	}
	if len(orig.Lines) == 0 {
		return nil, nil
	}

	entryDate := time.Now()
	lines := make([]domain.JournalLine, 0, len(orig.Lines))
	for _, l := range orig.Lines {
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   l.AccountID,
			Description: l.Description,
			Debit:       l.Credit,
			Credit:      l.Debit,
		})
	}

	desc := reason
	if desc == "" {
		desc = fmt.Sprintf("Pembatalan penjualan %s", sale.SaleNumber)
	}
	reversalOf := sale.JournalEntryID
	return u.finalizeSaleJournalPlan(ctx, sale, entryDate, lines, reversalOf, actorID)
}

// finalizeSaleJournalPlan memvalidasi akun, memastikan periode terbuka, dan
// menjumlahkan total debit/kredit rencana jurnal.
func (u *journalUsecase) finalizeSaleJournalPlan(ctx context.Context, sale *domain.Sale, entryDate time.Time, lines []domain.JournalLine, reversalOf *int64, actorID int64) (*domain.SaleJournalPlan, error) {
	var totalDebit, totalCredit float64
	for i := range lines {
		lines[i].LineNo = i + 1
		totalDebit += lines[i].Debit
		totalCredit += lines[i].Credit
	}
	totalDebit = round2(totalDebit)
	totalCredit = round2(totalCredit)
	if totalDebit <= 0 && totalCredit <= 0 {
		return nil, nil
	}
	if totalDebit != totalCredit {
		return nil, fmt.Errorf("%w: penjualan %s (debit %.2f != kredit %.2f)", domain.ErrJournalUnbalanced, sale.SaleNumber, totalDebit, totalCredit)
	}

	if _, err := u.validatePostableAccounts(ctx, toLineInputs(&lines)); err != nil {
		return nil, err
	}
	if _, err := u.periodRepo.EnsureOpen(ctx, entryDate.Year(), int(entryDate.Month()), actorID); err != nil {
		return nil, err
	}

	desc := fmt.Sprintf("Penjualan %s", saleLabel(sale))
	if reversalOf != nil {
		desc = fmt.Sprintf("Pembatalan penjualan %s", sale.SaleNumber)
	}
	createdBy := actorID
	return &domain.SaleJournalPlan{
		EntryDate:       entryDate,
		Description:     desc,
		SourceType:      domain.JournalSourceSale,
		SourceReference: sale.SaleNumber,
		ReversalOfID:    reversalOf,
		Lines:           lines,
		TotalDebit:      totalDebit,
		TotalCredit:     totalCredit,
		CreatedBy:       &createdBy,
	}, nil
}

func paymentMethodLabel(p domain.SalePayment) string {
	if p.PaymentMethodName != "" {
		return p.PaymentMethodName
	}
	return fmt.Sprintf("#%d", p.PaymentMethodID)
}

// saleLabel memakai nama member bila ada, atau "Umum" untuk penjualan tanpa member.
func saleLabel(sale *domain.Sale) string {
	if sale.MemberName != "" {
		return sale.MemberName
	}
	return "Umum"
}
