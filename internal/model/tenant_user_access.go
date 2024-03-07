package model

import (
	"time"

	"github.com/uptrace/bun"
)

type TenantUserAccess struct {
	bun.BaseModel `bun:"table:tenant_user_access,alias:tua"`

	ID        int       `bun:"id,pk,autoincrement"`
	TenantID  int       `bun:"tenant_id"`
	UserID    int       `bun:"user_id"`
	IsAdmin   bool      `bun:"is_admin,default:false"`
	CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:now()"`

	Tenant *Tenant `bun:"rel:belongs-to,join:tenant_id=id"`
	User   *User   `bun:"rel:belongs-to,join:user_id=id"`
}
