package xterrafile

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadCredentials(t *testing.T) {
	client := getStorageAccountsClient(true, "123")
	assert.True(t, client.SubscriptionID == "123")
}
