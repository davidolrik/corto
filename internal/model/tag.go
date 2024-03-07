package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Tag struct {
	bun.BaseModel `bun:"table:tags,alias:tg"`

	ID          int       `bun:"id,pk,autoincrement"`
	PublicID    string    `bun:"public_id"`
	TenantID    int       `bun:"tenant_id"`
	Slug        string    `bun:"slug"`
	Color       string    `bun:"color"` // Hex color like #4f46e5, empty for default
	Description string    `bun:"description"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()"`

	Tenant     *Tenant      `bun:"rel:belongs-to,join:tenant_id=id"`
	ShortCodes []*ShortCode `bun:"m2m:short_code_tags,join:Tag=ShortCode"`
}
