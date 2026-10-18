package domain

import "context"

type Role struct {
	ID          int32    `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type RoleCursorParams struct {
	Search    string
	CursorID  int32
	Direction string // "next" or "prev"
	Limit     int
}

type RoleCursorResult struct {
	Roles      []Role `json:"roles"`
	NextCursor *int32 `json:"next_cursor"`
	PrevCursor *int32 `json:"prev_cursor"`
	HasNext    bool   `json:"has_next"`
	HasPrev    bool   `json:"has_prev"`
	Total      int64  `json:"total"`
}

type RoleRepository interface {
	// 🌟 Hanya butuh 1 fungsi tunggal untuk menarik semua komponen data!
	GetAllRolesMaster(ctx context.Context, search string, offset, limit int) ([]Role, int64, error)
	GetRolesCursor(ctx context.Context, params RoleCursorParams) (*RoleCursorResult, error)
	GetAllMasterPermissions(ctx context.Context) ([]string, error)
	CreateRoleWithPermissions(ctx context.Context, name string, permissions []string) error
	UpdateRoleWithPermissions(ctx context.Context, id int32, name string, permissions []string) error
	DeleteRoleWithPermissions(ctx context.Context, id int32) error
}

type RoleUsecase interface {
	GetAllRolesMaster(ctx context.Context, search string, page, limit int) ([]Role, int64, error)
	GetRolesCursor(ctx context.Context, params RoleCursorParams) (*RoleCursorResult, error)
	GetAllMasterPermissions(ctx context.Context) ([]string, error)
	CreateRoleWithPermissions(ctx context.Context, name string, permissions []string) error
	UpdateRoleWithPermissions(ctx context.Context, id int32, name string, permissions []string) error
	DeleteRoleWithPermissions(ctx context.Context, id int32) error
}
