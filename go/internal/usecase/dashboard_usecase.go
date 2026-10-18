package usecase

import (
	"context"
	"go_backend/internal/domain"
)

// dashboardListLimit membatasi jumlah baris per daftar ringkas pada dashboard.
const dashboardListLimit = 5

type dashboardUsecase struct {
	repo domain.DashboardRepository
}

func NewDashboardUsecase(repo domain.DashboardRepository) domain.DashboardUsecase {
	return &dashboardUsecase{repo: repo}
}

func (u *dashboardUsecase) GetOverview(ctx context.Context) (*domain.DashboardOverview, error) {
	stats, err := u.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	lowStock, err := u.repo.GetLowStock(ctx, dashboardListLimit)
	if err != nil {
		return nil, err
	}

	recentPOs, err := u.repo.GetRecentPOs(ctx, dashboardListLimit)
	if err != nil {
		return nil, err
	}

	recentGRNs, err := u.repo.GetRecentGRNs(ctx, dashboardListLimit)
	if err != nil {
		return nil, err
	}

	return &domain.DashboardOverview{
		Stats:      stats,
		LowStock:   lowStock,
		RecentPOs:  recentPOs,
		RecentGRNs: recentGRNs,
	}, nil
}
