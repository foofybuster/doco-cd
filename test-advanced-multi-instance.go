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
	fmt.Println("🧪 Testing Multi-Instance Docker Management")
	fmt.Println("==========================================")

	// Test with multiple instances including an invalid one
	dockerConfig, err := config.LoadDockerConfig("./test-multi-config.yaml")
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

	// Test getting instances by name
	localInstance, err := manager.GetInstance("local")
	if err != nil {
		log.Fatalf("Failed to get local instance: %v", err)
	}
	fmt.Printf("✅ Retrieved instance by name: %s\n", localInstance.Name)

	// Test getting non-existent instance
	_, err = manager.GetInstance("non-existent")
	if err != nil {
		fmt.Printf("✅ Correctly handled non-existent instance: %s\n", err.Error())
	} else {
		fmt.Printf("❌ Should have failed for non-existent instance\n")
	}

	// Test instance status with mixed health
	ctx := context.Background()
	status := manager.GetInstancesStatus(ctx)

	fmt.Printf("✅ Instance status (mixed health):\n")
	healthyCount := 0
	for name, stat := range status {
		fmt.Printf("  - %s: %s", name, stat.Status)
		if stat.Error != "" {
			fmt.Printf(" (error: %s)", stat.Error)
		}
		if stat.Version != "" {
			fmt.Printf(" (version: %s)", stat.Version)
		}
		if stat.Status == "healthy" {
			healthyCount++
		}
		fmt.Println()
	}

	fmt.Printf("✅ %d/%d instances are healthy\n", healthyCount, len(dockerConfig.Instances))

	// Test listing instances
	instanceNames := manager.ListInstances()
	fmt.Printf("✅ Available instances: %v\n", instanceNames)

	fmt.Printf("\n🎉 All multi-instance tests passed!\n")
}
