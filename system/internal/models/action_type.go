package models

type ActionTypeEntry struct {
	Name        string `gorm:"primaryKey;size:20" json:"name"`
	Description string `gorm:"size:255" json:"description"`
	IsSecurity  bool   `gorm:"not null;default:false" json:"is_security"`
}
