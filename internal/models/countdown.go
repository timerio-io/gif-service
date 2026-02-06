package models

import (
	"time"
)

type CountdownType string

const (
	CountdownTypeEvent    CountdownType = "fixed"
	CountdownTypeBirthday CountdownType = "on_send"
	CountdownTypeHoliday  CountdownType = "on_open"
)

type Countdown struct {
	ID            string        `gorm:"primaryKey;type:text" json:"id"`
	UserID        string        `gorm:"not null;type:text;index" json:"user_id"`
	Name          string        `gorm:"not null;type:text" json:"name"`
	Type          CountdownType `gorm:"not null;type:text;check:type IN ('fixed','on_send','on_open');index" json:"type"`
	EndTime       *time.Time    `json:"end_time,omitempty"`
	Duration      *int          `json:"duration,omitempty"`
	StartedAt     *time.Time    `json:"started_at,omitempty"`
	PreviewURL    string        `gorm:"type:text" json:"preview_url"`
	Views         int           `gorm:"default:0" json:"views"`
	CreatedAt     time.Time     `json:"created_at"`
	IsSoftDeleted bool          `gorm:"default:false" json:"is_soft_deleted"`
}

type Template struct {
	ID          string    `gorm:"primaryKey;type:text" json:"id"`
	CountdownID string    `gorm:"not null;uniqueIndex;type:text" json:"countdown_id"`
	DesignID    string    `gorm:"not null;type:text" json:"design_id"`
	Name        string    `gorm:"not null;type:text" json:"name"`
	FontFamily  string    `gorm:"not null;type:text" json:"font_family"`
	FontSize    int       `gorm:"not null" json:"font_size"`
	BgColor     string    `gorm:"not null;type:text" json:"bg_color"`
	TextColor   string    `gorm:"not null;type:text" json:"text_color"`
	Layout      string    `gorm:"not null;type:text" json:"layout"`
	ShowUnits   bool      `gorm:"not null;default:1" json:"show_units"`
	CreatedAt   time.Time `json:"created_at"`

	Countdown Countdown `gorm:"foreignKey:CountdownID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type CountdownOpen struct {
	ID            string    `gorm:"primaryKey;type:text" json:"id"`
	CountdownID   string    `gorm:"not null;type:text;index" json:"countdown_id"`
	RecipientUID  string    `gorm:"not null;type:text;uniqueIndex:idx_countdown_recipient" json:"recipient_uid"`
	FirstOpenedAt time.Time `json:"first_opened_at"`

	Countdown Countdown `gorm:"foreignKey:CountdownID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

type ColorPalette struct {
	ID             string    `gorm:"primaryKey;type:text" json:"id"`
	UserID         string    `gorm:"not null;type:text;index" json:"user_id"`
	Name           string    `gorm:"not null;type:text" json:"name"`
	PrimaryColor   string    `gorm:"not null;type:text" json:"primary_color"`
	SecondaryColor string    `gorm:"not null;type:text" json:"secondary_color"`
	AccentColor    string    `gorm:"not null;type:text" json:"accent_color"`
	CreatedAt      time.Time `json:"created_at"`
}
