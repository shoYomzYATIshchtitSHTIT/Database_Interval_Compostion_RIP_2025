package ds

import (
	"database/sql"
	"time"
)

type Composition struct {
	ID          uint      `gorm:"primaryKey;autoIncrement;not null"`
	Status      string    `gorm:"type:varchar(15); not null"`
	CreatorID   uint      `gorm:"type:integer(15); not null"`
	ModeratorID *uint     `gorm:"type:integer(15)"`
	DateCreate  time.Time `gorm:"not null"`
	DateUpdate  time.Time
	DateFinish  sql.NullTime
	Belonging   string  `gorm:"type:varchar(30)"`
	Title       *string `gorm:"type:varchar(255)"`

	Creator             Users                `gorm:"foreignKey:CreatorID"`
	Moderator           Users                `gorm:"foreignKey:ModeratorID"`
	CompositorIntervals []CompositorInterval `gorm:"foreignKey:CompositionID"`
}
