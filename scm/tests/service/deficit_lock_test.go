package service_test

import (
	"context"
	"testing"
	"zeus-scm-service/internal/consumer"

	"github.com/stretchr/testify/assert"
)

func TestDeficitLockManager_LockDeficit_RabbitMQUnavailable(t *testing.T) {
	mgr := consumer.NewDeficitLockManager("")
	err := mgr.LockDeficit(context.Background(), "SOME-SKU", 10)
	assert.Error(t, err)
}
