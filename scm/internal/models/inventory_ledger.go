package models

import (
	"time"
)

type LedgerTxnType string

const (
	LedgerTxnTypeIN  LedgerTxnType = "IN"
	LedgerTxnTypeOUT LedgerTxnType = "OUT"
	LedgerTxnTypeADJ LedgerTxnType = "ADJ"
)

type LedgerRefType string

const (
	LedgerRefGoodsReceipt LedgerRefType = "goods_receipt"
	LedgerRefShipment     LedgerRefType = "shipment"
	LedgerRefAdjustment   LedgerRefType = "adjustment"
	LedgerRefInitial      LedgerRefType = "initial"
)

type InventoryLedger struct {
	ID             string        `gorm:"type:varchar(64);primary_key" json:"id"`
	SKU            string        `gorm:"type:varchar(100);not null;index" json:"sku"`
	Type           LedgerTxnType `gorm:"type:varchar(3);not null" json:"type"`
	QtyChange      int           `gorm:"not null" json:"qty_change"`
	RunningBalance int           `gorm:"not null" json:"running_balance"`
	Location       string        `gorm:"type:varchar(255);not null;default:'WH-A'" json:"location"`
	OperatorID     string        `gorm:"type:varchar(64);not null" json:"operator_id"`
	Reference      string        `gorm:"type:varchar(255);not null" json:"reference"`
	ReferenceType  LedgerRefType `gorm:"type:varchar(20);not null" json:"reference_type"`
	ReferenceID    string        `gorm:"type:varchar(64);not null" json:"reference_id"`
	CreatedAt      time.Time     `gorm:"autoCreateTime" json:"created_at"`
}
