package model

import (
	"time"

	"github.com/uptrace/bun"
)

type ShortCode struct {
	bun.BaseModel `bun:"table:short_codes,alias:sc"`

	ID           int        `bun:"id,pk,autoincrement"`
	PublicID     string     `bun:"public_id"`
	Title        string     `bun:"title"`
	Description  string     `bun:"description"`
	Slug         string     `bun:"slug"`
	TargetURL    string     `bun:"target_url"`
	FallbackURL  string     `bun:"fallback_url"`  // When url is not valid
	IsCrawlable  bool       `bun:"is_crawlable,default:false"` // Include in robots.txt
	ForwardQuery bool       `bun:"forward_query,default:false"` // Forward query string to target
	ValidSince   *time.Time `bun:"valid_since"`
	ValidUntil   *time.Time `bun:"valid_until"`
	MaxVisits    *int       `bun:"max_visits"` // nil means unlimited
	CreatedAt    time.Time  `bun:"created_at,notnull,default:now()"`
	UpdatedAt    time.Time  `bun:"updated_at,notnull,default:now()"`

	PlatformURLs []*PlatformSpecificURL `bun:"rel:has-many,join:id=short_code_id"`
	ShortURLs    []*ShortURL            `bun:"rel:has-many,join:id=short_code_id"`
	Tags         []*Tag                 `bun:"m2m:short_code_tags,join:ShortCode=Tag"`
}
