package usecase

import (
	"context"
	"strings"
	"testing"

	"go_backend/internal/domain"
)

type fakeAccountingSettingsAdminRepo struct {
	details []domain.AccountingSettingDetail
	upserts []domain.AccountingSetting
	actorID int64
	err     error
}

func (f *fakeAccountingSettingsAdminRepo) ListDetails(context.Context) ([]domain.AccountingSettingDetail, error) {
	return f.details, f.err
}

func (f *fakeAccountingSettingsAdminRepo) UpsertMany(_ context.Context, settings []domain.AccountingSetting, actorID int64) error {
	f.upserts = append([]domain.AccountingSetting(nil), settings...)
	f.actorID = actorID
	return f.err
}

func TestUpdateAccountingSettingsMenormalkanKunciDanMenyimpan(t *testing.T) {
	repo := &fakeAccountingSettingsAdminRepo{
		details: []domain.AccountingSettingDetail{
			{MappingKey: "SALES_REVENUE", AccountID: 4, AccountCode: "4-1100", AccountName: "Pendapatan Penjualan", AccountActive: true},
		},
	}
	coa := &fakeAccountRepo{accounts: map[int64]domain.ChartOfAccount{
		4: {ID: 4, Code: "4-1100", AccountType: domain.AccountTypeRevenue, IsActive: true},
	}}
	uc := NewAccountingSettingsUsecase(repo, coa)

	out, err := uc.UpdateSettings(context.Background(), []domain.AccountingSettingUpdateInput{
		{MappingKey: " sales_revenue ", AccountID: 4},
	}, 7)
	if err != nil {
		t.Fatalf("UpdateSettings gagal: %v", err)
	}
	if len(repo.upserts) != 1 {
		t.Fatalf("harus menyimpan 1 pengaturan, dapat %d", len(repo.upserts))
	}
	if repo.upserts[0].MappingKey != "SALES_REVENUE" {
		t.Fatalf("kunci harus dinormalkan menjadi SALES_REVENUE, dapat %q", repo.upserts[0].MappingKey)
	}
	if repo.actorID != 7 {
		t.Fatalf("updated_by harus 7, dapat %d", repo.actorID)
	}
	if len(out) != 1 || out[0].AccountCode != "4-1100" {
		t.Fatalf("hasil harus memuat detail akun terbaru, dapat %+v", out)
	}
}

func TestUpdateAccountingSettingsMenolakInputTidakValid(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		actorID int64
		input   []domain.AccountingSettingUpdateInput
		account domain.ChartOfAccount
		want    string
	}{
		{
			name:    "kunci tidak dikenal",
			actorID: 1,
			input:   []domain.AccountingSettingUpdateInput{{MappingKey: "UNKNOWN", AccountID: 4}},
			want:    "tidak dikenal",
		},
		{
			name:    "akun header",
			actorID: 1,
			input:   []domain.AccountingSettingUpdateInput{{MappingKey: "SALES_REVENUE", AccountID: 4}},
			account: domain.ChartOfAccount{ID: 4, Code: "4-0000", IsHeader: true, IsActive: true},
			want:    "akun header",
		},
		{
			name:    "akun nonaktif",
			actorID: 1,
			input:   []domain.AccountingSettingUpdateInput{{MappingKey: "SALES_REVENUE", AccountID: 4}},
			account: domain.ChartOfAccount{ID: 4, Code: "4-1100", IsActive: false},
			want:    "nonaktif",
		},
		{
			name:    "tanpa user",
			actorID: 0,
			input:   []domain.AccountingSettingUpdateInput{{MappingKey: "SALES_REVENUE", AccountID: 4}},
			want:    "tidak terautentikasi",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeAccountingSettingsAdminRepo{}
			accounts := map[int64]domain.ChartOfAccount{}
			if tc.account.ID > 0 {
				accounts[tc.account.ID] = tc.account
			}
			uc := NewAccountingSettingsUsecase(repo, &fakeAccountRepo{accounts: accounts})

			if _, err := uc.UpdateSettings(ctx, tc.input, tc.actorID); err == nil {
				t.Fatalf("harus gagal, tapi berhasil")
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("pesan error harus memuat %q, dapat %q", tc.want, err.Error())
			}
			if len(repo.upserts) != 0 {
				t.Fatalf("tidak boleh ada yang tersimpan saat validasi gagal")
			}
		})
	}
}
