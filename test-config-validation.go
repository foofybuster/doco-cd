package main

import (
	"fmt"
	"log"

	"github.com/kimdre/doco-cd/internal/config"
)

func main() {
	fmt.Println("🧪 Testing Docker Configuration Validation")
	fmt.Println("==========================================")

	// Test 1: Valid configuration
	validConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{Name: "local", Host: "unix:///var/run/docker.sock", Default: true},
			{Name: "remote", Host: "tcp://remote:2376", Default: false},
		},
	}
	
	err := validConfig.Validate()
	if err == nil {
		fmt.Printf("✅ Valid configuration passed validation\n")
	} else {
		fmt.Printf("❌ Valid configuration failed: %s\n", err.Error())
	}

	// Test 2: No instances
	emptyConfig := &config.DockerConfig{Instances: []config.DockerInstance{}}
	err = emptyConfig.Validate()
	if err != nil {
		fmt.Printf("✅ Empty config correctly rejected: %s\n", err.Error())
	} else {
		fmt.Printf("❌ Empty config should have been rejected\n")
	}

	// Test 3: Duplicate names
	duplicateConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{Name: "test", Host: "unix:///var/run/docker.sock", Default: true},
			{Name: "test", Host: "tcp://remote:2376", Default: false},
		},
	}
	err = duplicateConfig.Validate()
	if err != nil {
		fmt.Printf("✅ Duplicate names correctly rejected: %s\n", err.Error())
	} else {
		fmt.Printf("❌ Duplicate names should have been rejected\n")
	}

	// Test 4: No default instance
	noDefaultConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{Name: "test1", Host: "unix:///var/run/docker.sock", Default: false},
			{Name: "test2", Host: "tcp://remote:2376", Default: false},
		},
	}
	err = noDefaultConfig.Validate()
	if err != nil {
		fmt.Printf("✅ No default instance correctly rejected: %s\n", err.Error())
	} else {
		fmt.Printf("❌ No default instance should have been rejected\n")
	}

	// Test 5: Multiple defaults
	multiDefaultConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{Name: "test1", Host: "unix:///var/run/docker.sock", Default: true},
			{Name: "test2", Host: "tcp://remote:2376", Default: true},
		},
	}
	err = multiDefaultConfig.Validate()
	if err != nil {
		fmt.Printf("✅ Multiple defaults correctly rejected: %s\n", err.Error())
	} else {
		fmt.Printf("❌ Multiple defaults should have been rejected\n")
	}

	// Test 6: Empty name
	emptyNameConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{Name: "", Host: "unix:///var/run/docker.sock", Default: true},
		},
	}
	err = emptyNameConfig.Validate()
	if err != nil {
		fmt.Printf("✅ Empty name correctly rejected: %s\n", err.Error())
	} else {
		fmt.Printf("❌ Empty name should have been rejected\n")
	}

	// Test 7: Empty host
	emptyHostConfig := &config.DockerConfig{
		Instances: []config.DockerInstance{
			{Name: "test", Host: "", Default: true},
		},
	}
	err = emptyHostConfig.Validate()
	if err != nil {
		fmt.Printf("✅ Empty host correctly rejected: %s\n", err.Error())
	} else {
		fmt.Printf("❌ Empty host should have been rejected\n")
	}

	fmt.Printf("\n🎉 All validation tests passed!\n")
}
