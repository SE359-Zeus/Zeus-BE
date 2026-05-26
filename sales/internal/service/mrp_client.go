package service

import (
	"context"
	"time"
)

type MRPCreateOrderReq struct {
	ProductModelCode string
	TargetQuantity   int
	ScheduledAt      time.Time
}

type MRPClient interface {
	CreateProductionOrder(ctx context.Context, req MRPCreateOrderReq) error
}
