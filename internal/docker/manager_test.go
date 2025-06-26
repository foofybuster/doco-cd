package docker

import (
	"context"
	"testing"

	"github.com/kimdre/doco-cd/internal/config"
	"github.com/kimdre/doco-cd/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientManager(t *testing.T) {
	// Skip if Docker is not available
	err := VerifySocketConnection()
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}

	dockerConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{
				Name:    "local",
				Host:    "unix:///var/run/docker.sock",
				Default: true,
			},
		},
	}

	log := logger.New(12)

	manager, err := NewClientManager(dockerConfig, false, false, log.Logger)
	require.NoError(t, err)
	defer manager.Close()

	// Test getting default instance
	instance, err := manager.GetDefaultInstance()
	require.NoError(t, err)
	assert.Equal(t, "local", instance.Name)

	// Test getting instance by name
	instance, err = manager.GetInstance("local")
	require.NoError(t, err)
	assert.Equal(t, "local", instance.Name)

	// Test getting non-existent instance
	_, err = manager.GetInstance("non-existent")
	assert.Error(t, err)

	// Test listing instances
	instances := manager.ListInstances()
	assert.Contains(t, instances, "local")

	// Test verifying connections
	ctx := context.Background()
	err = manager.VerifyAllConnections(ctx)
	assert.NoError(t, err)

	// Test getting instances status
	status := manager.GetInstancesStatus(ctx)
	assert.Contains(t, status, "local")
	assert.Equal(t, "healthy", status["local"].Status)
}

func TestClientManagerWithMultipleInstances(t *testing.T) {
	// Skip if Docker is not available
	err := VerifySocketConnection()
	if err != nil {
		t.Skip("Docker not available, skipping test")
	}

	dockerConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{
				Name:    "local",
				Host:    "unix:///var/run/docker.sock",
				Default: true,
			},
			{
				Name: "invalid",
				Host: "tcp://invalid-host:2376",
			},
		},
	}

	log := logger.New(12)

	manager, err := NewClientManager(dockerConfig, false, false, log.Logger)
	require.NoError(t, err)
	defer manager.Close()

	// Test getting instances status with mixed health
	ctx := context.Background()
	status := manager.GetInstancesStatus(ctx)
	
	assert.Contains(t, status, "local")
	assert.Equal(t, "healthy", status["local"].Status)
	
	assert.Contains(t, status, "invalid")
	assert.Equal(t, "unhealthy", status["invalid"].Status)
	assert.NotEmpty(t, status["invalid"].Error)
}
