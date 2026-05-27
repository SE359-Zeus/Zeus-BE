package service

import (
	infraCache "zeus-sales-service/internal/infrastructure/cache"
	infraMessaging "zeus-sales-service/internal/infrastructure/messaging"
	"zeus-sales-service/internal/repository"
)

type Infrastructure struct {
	Cache     *infraCache.Store
	Publisher infraMessaging.Publisher
	SCMClient SCMClient
	MRPClient MRPClient
}

func NewInfrastructure(cache *infraCache.Store, publisher infraMessaging.Publisher, scmClient SCMClient, mrpClient MRPClient) *Infrastructure {
	return &Infrastructure{Cache: cache, Publisher: publisher, SCMClient: scmClient, MRPClient: mrpClient}
}

type Services struct {
	Clients     *ClientService
	Orders      *OrderService
	Fulfillment *FulfillmentService
	Infra       *Infrastructure
}

func NewServices(sqliteRepo repository.DbRepository, valkeyRepo repository.CacheRepository, infra ...*Infrastructure) *Services {
	var sharedInfra *Infrastructure
	if len(infra) > 0 {
		sharedInfra = infra[0]
	}
	clients := NewClientService(sqliteRepo, valkeyRepo, sharedInfra)
	orders := NewOrderService(sqliteRepo, valkeyRepo, clients, sharedInfra)
	fulfillment := NewFulfillmentService(sqliteRepo, valkeyRepo, sharedInfra)
	return &Services{
		Clients:     clients,
		Orders:      orders,
		Fulfillment: fulfillment,
		Infra:       sharedInfra,
	}
}
