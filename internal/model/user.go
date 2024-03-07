package model

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        int       `bun:"id,pk,autoincrement"`
	PublicID  string    `bun:"public_id,notnull,unique"`
	Username  string    `bun:"username"`
	Password  string    `bun:"password"`
	CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:now()"`

	Tenants []*Tenant `bun:"rel:has-many,join:id=owner_id"`
}
