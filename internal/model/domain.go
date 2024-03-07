package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Domain struct {
	bun.BaseModel `bun:"table:domains,alias:d"`

	ID          int       `bun:"id,pk,autoincrement"`
	PublicID    string    `bun:"public_id"`
	TenantID    int       `bun:"tenant_id"`
	FQDN        string    `bun:"fqdn,notnull,unique"`
	FallbackURL string    `bun:"fallback_url"`
	Description string    `bun:"description"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()"`

	Tenant    *Tenant     `bun:"rel:belongs-to,join:tenant_id=id"`
	ShortURLs []*ShortURL `bun:"rel:has-many,join:id=domain_id"`
}
