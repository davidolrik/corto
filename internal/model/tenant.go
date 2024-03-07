package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Tenant struct {
	bun.BaseModel `bun:"table:tenants,alias:t"`

	ID        int       `bun:"id,pk,autoincrement"`
	PublicID  string    `bun:"public_id,notnull,unique"`
	OwnerID   int       `bun:"owner_id"`
	Slug      string    `bun:"slug"`
	Name      string    `bun:"name"`
	CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:now()"`

	Owner   *User              `bun:"rel:belongs-to,join:owner_id=id"`
	Domains []*Domain          `bun:"rel:has-many,join:id=tenant_id"`
	Tags    []*Tag             `bun:"rel:has-many,join:id=tenant_id"`
	Users   []*User            `bun:"m2m:tenant_user_access,join:Tenant=User"`
	Access  []*TenantUserAccess `bun:"rel:has-many,join:id=tenant_id"`
}
