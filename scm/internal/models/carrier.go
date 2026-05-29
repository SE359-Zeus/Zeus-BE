package models

// Carrier represents a shipping carrier available for selection in the UI.
type Carrier struct {
	ID   int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:text;uniqueIndex;not null"  json:"name"`
	Code string `gorm:"type:text;uniqueIndex;not null"  json:"code"`
}
