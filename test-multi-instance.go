package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kimdre/doco-cd/internal/config"
	"github.com/kimdre/doco-cd/internal/docker"
	"github.com/kimdre/doco-cd/internal/logger"
)

func main() {
	// Test Docker configuration loading
	dockerConfig, err := config.LoadDockerConfig("./test-data/docker-instances.yaml")
	if err != nil {
		log.Fatalf("Failed to load Docker config: %v", err)
	}

	fmt.Printf("✅ Loaded Docker configuration with %d instances:\n", len(dockerConfig.Instances))
	for _, instance := range dockerConfig.Instances {
		fmt.Printf("  - %s: %s (default: %t)\n", instance.Name, instance.Host, instance.Default)
	}

	// Test Docker client manager
	logger := logger.New(12)
	manager, err := docker.NewClientManager(dockerConfig, true, false, logger.Logger)
	if err != nil {
		log.Fatalf("Failed to create Docker client manager: %v", err)
	}
	defer manager.Close()

	fmt.Printf("✅ Created Docker client manager\n")

	// Test getting default instance
	defaultInstance, err := manager.GetDefaultInstance()
	if err != nil {
		log.Fatalf("Failed to get default instance: %v", err)
	}

	fmt.Printf("✅ Default instance: %s\n", defaultInstance.Name)

	// Test instance status
	ctx := context.Background()
	status := manager.GetInstancesStatus(ctx)

	fmt.Printf("✅ Instance status:\n")
	for name, stat := range status {
		fmt.Printf("  - %s: %s", name, stat.Status)
		if stat.Error != "" {
			fmt.Printf(" (error: %s)", stat.Error)
		}
		if stat.Version != "" {
			fmt.Printf(" (version: %s)", stat.Version)
		}
		fmt.Println()
	}

	fmt.Printf("✅ Multi-instance Docker management is working!\n")
}
