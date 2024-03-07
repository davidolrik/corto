package model

import (
	"time"

	"github.com/uptrace/bun"
)

type ShortURL struct {
	bun.BaseModel `bun:"table:short_urls,alias:su"`

	ID          int       `bun:"id,pk,autoincrement"`
	PublicID    string    `bun:"public_id"`
	DomainID    int       `bun:"domain_id"`
	ShortCodeID int       `bun:"short_code_id"`
	Slug        string    `bun:"slug"` // Kept in sync with the short code's slug; unique per domain
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()"`

	Domain    *Domain    `bun:"rel:belongs-to,join:domain_id=id"`
	ShortCode *ShortCode `bun:"rel:belongs-to,join:short_code_id=id"`
	Visits    []*Visit   `bun:"rel:has-many,join:id=short_url_id"`
}
