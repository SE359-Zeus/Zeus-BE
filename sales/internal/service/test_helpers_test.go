package service

func newTestServicesWithMocks(db *MockDbRepository, cache *MockCacheRepository) *Services {
	return NewServices(db, cache)
}
