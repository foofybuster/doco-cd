package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDockerConfig(t *testing.T) {
	// Test with non-existent file (should return default config)
	config, err := LoadDockerConfig("/non-existent/file.yaml")
	require.NoError(t, err)
	assert.Len(t, config.Instances, 1)
	assert.Equal(t, "local", config.Instances[0].Name)
	assert.True(t, config.Instances[0].Default)

	// Test with valid config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "docker-config.yaml")
	
	configContent := `instances:
  - name: "local"
    host: "unix:///var/run/docker.sock"
    default: true
  - name: "remote"
    host: "tcp://remote:2376"
    tls_verify: true
    tls_cert: "cert.pem"
    tls_key: "key.pem"
    tls_ca: "ca.pem"
`
	
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	
	config, err = LoadDockerConfig(configFile)
	require.NoError(t, err)
	assert.Len(t, config.Instances, 2)
	
	// Check local instance
	local := config.Instances[0]
	assert.Equal(t, "local", local.Name)
	assert.True(t, local.Default)
	
	// Check remote instance
	remote := config.Instances[1]
	assert.Equal(t, "remote", remote.Name)
	assert.False(t, remote.Default)
	assert.True(t, remote.TLSVerify)
	assert.Equal(t, filepath.Join(tmpDir, "cert.pem"), remote.TLSCert)
}

func TestDockerConfigValidation(t *testing.T) {
	// Test duplicate names
	config := &DockerConfig{
		Instances: []DockerInstance{
			{Name: "test", Host: "unix:///var/run/docker.sock", Default: true},
			{Name: "test", Host: "tcp://remote:2376", Default: false},
		},
	}
	
	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate instance name")
	
	// Test no default
	config = &DockerConfig{
		Instances: []DockerInstance{
			{Name: "test1", Host: "unix:///var/run/docker.sock", Default: false},
			{Name: "test2", Host: "tcp://remote:2376", Default: false},
		},
	}
	
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no default Docker instance")
	
	// Test multiple defaults
	config = &DockerConfig{
		Instances: []DockerInstance{
			{Name: "test1", Host: "unix:///var/run/docker.sock", Default: true},
			{Name: "test2", Host: "tcp://remote:2376", Default: true},
		},
	}
	
	err = config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple default Docker instances")
	
	// Test valid config
	config = &DockerConfig{
		Instances: []DockerInstance{
			{Name: "local", Host: "unix:///var/run/docker.sock", Default: true},
			{Name: "remote", Host: "tcp://remote:2376", Default: false},
		},
	}
	
	err = config.Validate()
	assert.NoError(t, err)
}

func TestGetInstance(t *testing.T) {
	config := &DockerConfig{
		Instances: []DockerInstance{
			{Name: "local", Host: "unix:///var/run/docker.sock", Default: true},
			{Name: "remote", Host: "tcp://remote:2376", Default: false},
		},
	}
	
	// Test existing instance
	instance, err := config.GetInstance("local")
	require.NoError(t, err)
	assert.Equal(t, "local", instance.Name)
	
	// Test non-existent instance
	_, err = config.GetInstance("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Docker instance not found")
	
	// Test default instance
	instance, err = config.GetDefaultInstance()
	require.NoError(t, err)
	assert.Equal(t, "local", instance.Name)
	assert.True(t, instance.Default)
}
