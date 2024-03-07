package model

import (
	"time"

	"github.com/uptrace/bun"
)

type ShortCodeTag struct {
	bun.BaseModel `bun:"table:short_code_tags,alias:sct"`

	ID          int       `bun:"id,pk,autoincrement"`
	TagID       int       `bun:"tag_id"`
	ShortCodeID int       `bun:"shortcode_id"` // Matches column name in migration
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()"`

	Tag       *Tag       `bun:"rel:belongs-to,join:tag_id=id"`
	ShortCode *ShortCode `bun:"rel:belongs-to,join:shortcode_id=id"`
}
