package service

import (
	"context"
	"testing"

	"zeus-sales-service/internal/models"
	rootrepo "zeus-sales-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClientService_ResolveOrCreateClient_CreatesWhenMissing(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Clients

	name := "New Client"
	db.On("GetClientByName", mock.Anything, name).Return(nil, rootrepo.ErrNotFound)
	db.On("CreateClient", mock.Anything, mock.AnythingOfType("*models.Client")).Return(nil)

	client, err := svc.ResolveOrCreateClient(context.Background(), name, "Addr 1", models.ClientTierB2C)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, name, client.Name)
	db.AssertExpectations(t)
}

func TestClientService_ResolveOrCreateClient_ExistingReturned(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Clients

	existing := &models.Client{ID: uuid.New(), Name: "Existing", Tier: models.ClientTierB2B}
	db.On("GetClientByName", mock.Anything, "Existing").Return(existing, nil)

	client, err := svc.ResolveOrCreateClient(context.Background(), "Existing", "", models.ClientTierB2B)
	require.NoError(t, err)
	assert.Equal(t, existing.ID, client.ID)
	db.AssertExpectations(t)
}

func TestClientService_ListClients_ReturnsList(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Clients

	clients := []models.Client{{ID: uuid.New(), Name: "A"}, {ID: uuid.New(), Name: "B"}}
	db.On("ListClients", mock.Anything).Return(clients, nil)

	res, err := svc.ListClients(context.Background())
	require.NoError(t, err)
	assert.Len(t, res, 2)
}

func TestClientService_UpdateClient_SuccessClearsQueue(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Clients

	id := uuid.New()
	client := &models.Client{ID: id, Name: "Old", Tier: models.ClientTierB2C}
	db.On("GetClient", mock.Anything, id).Return(client, nil)
	db.On("UpdateClient", mock.Anything, mock.AnythingOfType("*models.Client")).Return(nil)
	cache.On("ClearQueue", mock.Anything).Return(nil)

	req := models.UpdateClientRequest{Name: ptrString("New Name")}
	updated, err := svc.UpdateClient(context.Background(), id, req)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestClientService_UpdateClient_AllowsPartialPatch(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Clients

	id := uuid.New()
	client := &models.Client{ID: id, Name: "Old Name", Tier: models.ClientTierB2C, DefaultDestinationAddress: "Old Address"}
	db.On("GetClient", mock.Anything, id).Return(client, nil)
	db.On("UpdateClient", mock.Anything, mock.MatchedBy(func(updated *models.Client) bool {
		return updated.Name == "Old Name" && updated.Tier == models.ClientTierB2B && updated.DefaultDestinationAddress == "New Address"
	})).Return(nil)
	cache.On("ClearQueue", mock.Anything).Return(nil)

	tier := models.ClientTierB2B
	updated, err := svc.UpdateClient(context.Background(), id, models.UpdateClientRequest{
		Tier:                      &tier,
		DefaultDestinationAddress: ptrString("New Address"),
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Old Name", updated.Name)
	assert.Equal(t, models.ClientTierB2B, updated.Tier)
	assert.Equal(t, "New Address", updated.DefaultDestinationAddress)
	db.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestClientService_UpdateClient_NotFound(t *testing.T) {
	db := setupMockDbRepo()
	cache := setupMockCacheRepo()
	svc := newTestServicesWithMocks(db, cache).Clients

	id := uuid.New()
	db.On("GetClient", mock.Anything, id).Return(nil, rootrepo.ErrNotFound)

	_, err := svc.UpdateClient(context.Background(), id, models.UpdateClientRequest{Name: ptrString("X")})
	require.Error(t, err)
}

// ptrString helper intentionally omitted; reuse ptrString from other tests in package
