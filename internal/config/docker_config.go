package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DockerInstance represents a single Docker host configuration
type DockerInstance struct {
	Name      string            `yaml:"name"`                                 // Unique name for the Docker instance
	Host      string            `yaml:"host"`                                 // Docker host (tcp://host:port or unix:///path/to/socket)
	TLSCert   string            `yaml:"tls_cert,omitempty"`                   // Path to TLS certificate file
	TLSKey    string            `yaml:"tls_key,omitempty"`                    // Path to TLS key file
	TLSCA     string            `yaml:"tls_ca,omitempty"`                     // Path to TLS CA file
	TLSVerify bool              `yaml:"tls_verify,omitempty"`                 // Enable TLS verification
	Default   bool              `yaml:"default,omitempty"`                    // Mark this as the default instance
	Labels    map[string]string `yaml:"labels,omitempty"`                     // Optional labels for the instance
}

// DockerConfig represents the configuration for multiple Docker instances
type DockerConfig struct {
	Instances []DockerInstance `yaml:"instances"` // List of Docker instances
}

// Validate performs validation on the Docker configuration
func (dc *DockerConfig) Validate() error {
	if len(dc.Instances) == 0 {
		return fmt.Errorf("docker config validation failed: no instances configured")
	}

	// Check for duplicate names
	names := make(map[string]bool)
	defaultCount := 0

	for _, instance := range dc.Instances {
		if instance.Name == "" {
			return fmt.Errorf("docker config validation failed: instance name cannot be empty")
		}
		if instance.Host == "" {
			return fmt.Errorf("docker config validation failed: instance host cannot be empty")
		}

		if names[instance.Name] {
			return fmt.Errorf("duplicate instance name: %s", instance.Name)
		}
		names[instance.Name] = true

		if instance.Default {
			defaultCount++
		}
	}

	if defaultCount == 0 {
		return fmt.Errorf("no default Docker instance specified")
	}

	if defaultCount > 1 {
		return fmt.Errorf("multiple default Docker instances specified")
	}

	return nil
}

// GetDefaultInstance returns the default Docker instance
func (dc *DockerConfig) GetDefaultInstance() (*DockerInstance, error) {
	for _, instance := range dc.Instances {
		if instance.Default {
			return &instance, nil
		}
	}
	return nil, fmt.Errorf("no default Docker instance found")
}

// GetInstance returns a Docker instance by name
func (dc *DockerConfig) GetInstance(name string) (*DockerInstance, error) {
	for _, instance := range dc.Instances {
		if instance.Name == name {
			return &instance, nil
		}
	}
	return nil, fmt.Errorf("Docker instance not found: %s", name)
}

// LoadDockerConfig loads Docker configuration from a file
func LoadDockerConfig(configPath string) (*DockerConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default local configuration if file doesn't exist
		return &DockerConfig{
			Instances: []DockerInstance{
				{
					Name:    "local",
					Host:    "unix:///var/run/docker.sock",
					Default: true,
				},
			},
		}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Docker config file %s: %w", configPath, err)
	}

	var config DockerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Docker config file %s: %w", configPath, err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Docker config in %s: %w", configPath, err)
	}

	// Resolve relative paths for TLS certificates
	configDir := filepath.Dir(configPath)
	for i := range config.Instances {
		instance := &config.Instances[i]
		if instance.TLSCert != "" && !filepath.IsAbs(instance.TLSCert) {
			instance.TLSCert = filepath.Join(configDir, instance.TLSCert)
		}
		if instance.TLSKey != "" && !filepath.IsAbs(instance.TLSKey) {
			instance.TLSKey = filepath.Join(configDir, instance.TLSKey)
		}
		if instance.TLSCA != "" && !filepath.IsAbs(instance.TLSCA) {
			instance.TLSCA = filepath.Join(configDir, instance.TLSCA)
		}
	}

	return &config, nil
}

// SaveDockerConfig saves Docker configuration to a file
func SaveDockerConfig(config *DockerConfig, configPath string) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Docker config: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write Docker config file: %w", err)
	}

	return nil
}
