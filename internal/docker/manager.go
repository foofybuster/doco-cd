package docker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
	"github.com/kimdre/doco-cd/internal/config"
)

// InstanceClient represents a Docker client for a specific instance
type InstanceClient struct {
	Name      string
	Config    config.DockerInstance
	CLI       command.Cli
	APIClient *client.Client
	logger    *slog.Logger
}

// ClientManager manages multiple Docker instances
type ClientManager struct {
	instances       map[string]*InstanceClient
	defaultInstance string
	logger          *slog.Logger
}

// NewClientManager creates a new Docker client manager
func NewClientManager(dockerConfig *config.DockerConfig, quiet, skipTLSVerification bool, logger *slog.Logger) (*ClientManager, error) {
	manager := &ClientManager{
		instances: make(map[string]*InstanceClient),
		logger:    logger,
	}

	// Initialize each Docker instance
	for _, instanceConfig := range dockerConfig.Instances {
		instance, err := createInstanceClient(instanceConfig, quiet, skipTLSVerification, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance %s: %w", instanceConfig.Name, err)
		}

		manager.instances[instanceConfig.Name] = instance

		if instanceConfig.Default {
			manager.defaultInstance = instanceConfig.Name
		}

		logger.Debug("Docker instance initialized",
			slog.String("name", instanceConfig.Name),
			slog.String("host", instanceConfig.Host),
			slog.Bool("default", instanceConfig.Default),
		)
	}

	return manager, nil
}

// createInstanceClient creates a Docker client for a specific instance
func createInstanceClient(instanceConfig config.DockerInstance, quiet, skipTLSVerification bool, logger *slog.Logger) (*InstanceClient, error) {
	var (
		outputStream io.Writer
		errorStream  io.Writer
	)

	if quiet {
		outputStream = io.Discard
		errorStream = io.Discard
	} else {
		outputStream = os.Stdout
		errorStream = os.Stderr
	}

	// Create Docker CLI
	dockerCli, err := command.NewDockerCli(
		command.WithOutputStream(outputStream),
		command.WithErrorStream(errorStream),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker cli: %w", err)
	}

	// Configure client options
	opts := &flags.ClientOptions{
		LogLevel:  "error",
		TLSVerify: instanceConfig.TLSVerify && !skipTLSVerification,
	}

	// Set host or use default context
	if instanceConfig.Host != "" {
		opts.Hosts = []string{instanceConfig.Host}
	} else {
		opts.Context = "default"
	}

	// Set TLS options
	if instanceConfig.TLSVerify && !skipTLSVerification {
		opts.TLS = true
		opts.TLSVerify = true
		if instanceConfig.TLSCert != "" {
			opts.TLSOptions.CertFile = instanceConfig.TLSCert
		}
		if instanceConfig.TLSKey != "" {
			opts.TLSOptions.KeyFile = instanceConfig.TLSKey
		}
		if instanceConfig.TLSCA != "" {
			opts.TLSOptions.CAFile = instanceConfig.TLSCA
		}
	}

	// Initialize Docker CLI
	err = dockerCli.Initialize(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker cli: %w", err)
	}

	// Create API client with the same configuration
	clientOpts := []client.Opt{
		client.WithAPIVersionNegotiation(),
	}

	// Set host for API client
	if instanceConfig.Host != "" {
		clientOpts = append(clientOpts, client.WithHost(instanceConfig.Host))
	}

	// Configure TLS for API client
	if instanceConfig.TLSVerify && !skipTLSVerification {
		clientOpts = append(clientOpts, client.WithTLSClientConfig(
			instanceConfig.TLSCA,
			instanceConfig.TLSCert,
			instanceConfig.TLSKey,
		))
	}

	apiClient, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return &InstanceClient{
		Name:      instanceConfig.Name,
		Config:    instanceConfig,
		CLI:       dockerCli,
		APIClient: apiClient,
		logger:    logger.With(slog.String("docker_instance", instanceConfig.Name)),
	}, nil
}

// createTLSConfig creates a TLS configuration for Docker client
func createTLSConfig(instanceConfig config.DockerInstance) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !instanceConfig.TLSVerify,
	}

	if instanceConfig.TLSVerify {
		// Load client certificate
		if instanceConfig.TLSCert != "" && instanceConfig.TLSKey != "" {
			cert, err := tls.LoadX509KeyPair(instanceConfig.TLSCert, instanceConfig.TLSKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// Load CA certificate
		if instanceConfig.TLSCA != "" {
			caCert, err := os.ReadFile(instanceConfig.TLSCA)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}
	}

	return tlsConfig, nil
}

// GetInstance returns a Docker instance by name
func (cm *ClientManager) GetInstance(name string) (*InstanceClient, error) {
	if name == "" {
		return cm.GetDefaultInstance()
	}

	instance, exists := cm.instances[name]
	if !exists {
		return nil, fmt.Errorf("Docker instance not found: %s", name)
	}

	return instance, nil
}

// GetDefaultInstance returns the default Docker instance
func (cm *ClientManager) GetDefaultInstance() (*InstanceClient, error) {
	if cm.defaultInstance == "" {
		return nil, fmt.Errorf("no default Docker instance configured")
	}

	instance, exists := cm.instances[cm.defaultInstance]
	if !exists {
		return nil, fmt.Errorf("default Docker instance not found: %s", cm.defaultInstance)
	}

	return instance, nil
}

// ListInstances returns a list of all instance names
func (cm *ClientManager) ListInstances() []string {
	var names []string
	for name := range cm.instances {
		names = append(names, name)
	}
	return names
}

// GetInstancesStatus returns the status of all Docker instances
func (cm *ClientManager) GetInstancesStatus(ctx context.Context) map[string]InstanceStatus {
	status := make(map[string]InstanceStatus)

	for name, instance := range cm.instances {
		instanceStatus := InstanceStatus{
			Name: name,
			Host: instance.Config.Host,
		}

		// Try to ping the Docker daemon
		_, err := instance.APIClient.Ping(ctx)
		if err != nil {
			instanceStatus.Status = "unhealthy"
			instanceStatus.Error = err.Error()
		} else {
			instanceStatus.Status = "healthy"
			
			// Get Docker version
			version, err := instance.APIClient.ServerVersion(ctx)
			if err == nil {
				instanceStatus.Version = version.Version
			}
		}

		status[name] = instanceStatus
	}

	return status
}

// InstanceStatus represents the status of a Docker instance
type InstanceStatus struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Close closes all Docker clients
func (cm *ClientManager) Close() error {
	var errors []string

	for name, instance := range cm.instances {
		if err := instance.APIClient.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to close instance %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing instances: %s", strings.Join(errors, "; "))
	}

	return nil
}

// VerifyInstanceConnection verifies the connection to a specific Docker instance
func (cm *ClientManager) VerifyInstanceConnection(ctx context.Context, instanceName string) error {
	instance, err := cm.GetInstance(instanceName)
	if err != nil {
		return err
	}

	_, err = instance.APIClient.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Docker instance %s: %w", instanceName, err)
	}

	return nil
}

// VerifyAllConnections verifies connections to all Docker instances
func (cm *ClientManager) VerifyAllConnections(ctx context.Context) error {
	var errors []string

	for name := range cm.instances {
		if err := cm.VerifyInstanceConnection(ctx, name); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("connection verification failed: %s", strings.Join(errors, "; "))
	}

	return nil
}
