package service

import "zeus-sales-service/internal/repository"

type Services struct {
	Clients     *ClientService
	Orders      *OrderService
	Fulfillment *FulfillmentService
}

func NewServices(sqliteRepo repository.DbRepository, valkeyRepo repository.CacheRepository) *Services {
	clients := NewClientService(sqliteRepo, valkeyRepo)
	orders := NewOrderService(sqliteRepo, valkeyRepo, clients)
	fulfillment := NewFulfillmentService(sqliteRepo, valkeyRepo)
	return &Services{
		Clients:     clients,
		Orders:      orders,
		Fulfillment: fulfillment,
	}
}
