package models

type AccountStatusState struct {
	ID   int32         `gorm:"primaryKey;autoIncrement:false"`
	Name AccountStatus `gorm:"type:varchar(20);not null;uniqueIndex"`
}
