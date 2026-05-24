package service

import (
	"zeus-mrp-service/internal/repository"
)

type ProductionService struct {
	repo  repository.MRPRepository
	cache repository.CacheRepository
}

func NewProductionService(repo repository.MRPRepository, cache ...repository.CacheRepository) *ProductionService {
	var cacheRepo repository.CacheRepository
	if len(cache) > 0 {
		cacheRepo = cache[0]
	}

	return &ProductionService{repo: repo, cache: cacheRepo}
}
