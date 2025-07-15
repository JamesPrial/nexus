package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jamesprial/nexus/internal/config"
	"github.com/jamesprial/nexus/internal/container"
	"github.com/jamesprial/nexus/internal/gateway"
	"github.com/jamesprial/nexus/internal/logging"
)

// Build-time variables (set by ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func run() error {
	// Create dependency injection container
	cont := container.New()

	// Set up logger
	logger := logging.NewStandardLogger(logging.LevelInfo)
	cont.SetLogger(logger)

	// Set up configuration loader
	configPath := "config.yaml"
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		configPath = path
	}
	
	configLoader := config.NewFileLoader(configPath)
	cont.SetConfigLoader(configLoader)

	// Initialize all dependencies
	if err := cont.Initialize(); err != nil {
		return err
	}

	// Create gateway service
	gatewayService := gateway.NewService(cont)

	// Start the gateway
	if err := gatewayService.Start(); err != nil {
		return err
	}

	logger.Info("Nexus gateway started successfully", map[string]interface{}{
		"config_path": configPath,
	})

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutdown signal received", map[string]interface{}{})

	// Graceful shutdown
	return gatewayService.Stop()
}

func main() {
	// Parse command line flags
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("Nexus API Gateway\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Go Version: %s\n", "go1.23")
		return
	}

	// Show help and exit
	if *showHelp {
		fmt.Println("Nexus API Gateway - Self-hosted AI API Gateway")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("  %s [options]\n", os.Args[0])
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -help         Show help information")
		fmt.Println("  -version      Show version information")
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  CONFIG_PATH   Path to configuration file (default: config.yaml)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Printf("  %s                    # Start the gateway\n", os.Args[0])
		fmt.Printf("  CONFIG_PATH=/etc/nexus/config.yaml %s\n", os.Args[0])
		return
	}

	log.Printf("Nexus API Gateway %s (built %s)", Version, BuildTime)
	
	if err := run(); err != nil {
		log.Fatalf("failed to run gateway: %v", err)
	}
}