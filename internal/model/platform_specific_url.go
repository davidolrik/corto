package model

import (
	"time"

	"github.com/uptrace/bun"
)

type PlatformSpecificURL struct {
	bun.BaseModel `bun:"table:platform_specific_urls,alias:psu"`

	ID          int       `bun:"id,pk,autoincrement"`
	PublicID    string    `bun:"public_id"`
	ShortCodeID int       `bun:"short_code_id"`
	TargetURL   string    `bun:"target_url"`
	FallbackURL string    `bun:"fallback_url"` // When url is not valid
	Platform    string    `bun:"platform"`     // Mobile, iOS, Android, Desktop, macOS, Windows, Linux
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()"`

	ShortCode *ShortCode `bun:"rel:belongs-to,join:short_code_id=id"`
}
