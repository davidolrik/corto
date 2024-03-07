package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Visit struct {
	bun.BaseModel `bun:"table:visits,alias:v"`

	ID         int       `bun:"id,pk,autoincrement"`
	PublicID   string    `bun:"public_id"`
	ShortURLID int       `bun:"short_url_id"`
	IPAddress  string    `bun:"ip_address"`
	UserAgent  string    `bun:"user_agent"`
	Referer    string    `bun:"refere"`   // Intentional misspelling, matches HTTP RFC 2616
	Country    string    `bun:"country"`
	Campaign   string    `bun:"campaign"`
	CreatedAt  time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt  time.Time `bun:"updated_at,notnull,default:now()"`

	ShortURL *ShortURL `bun:"rel:belongs-to,join:short_url_id=id"`
}
