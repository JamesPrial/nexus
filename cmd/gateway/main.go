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

func mainWithDI() error {
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

	logger.Info("Nexus gateway started with dependency injection", map[string]interface{}{
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
		useDI       = flag.Bool("di", false, "Use dependency injection architecture")
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
		fmt.Println("  -di           Use dependency injection architecture")
		fmt.Println("  -help         Show help information")
		fmt.Println("  -version      Show version information")
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  CONFIG_PATH   Path to configuration file (default: config.yaml)")
		fmt.Println("  USE_DI        Use dependency injection architecture (default: false)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Printf("  %s                    # Start with legacy architecture\n", os.Args[0])
		fmt.Printf("  %s -di               # Start with dependency injection\n", os.Args[0])
		fmt.Printf("  USE_DI=true %s       # Start with DI via environment\n", os.Args[0])
		return
	}

	// Determine architecture to use
	useNewArch := *useDI || os.Getenv("USE_DI") == "true"
	
	log.Printf("Nexus API Gateway %s (built %s)", Version, BuildTime)
	
	if useNewArch {
		log.Println("Starting with dependency injection architecture...")
		if err := mainWithDI(); err != nil {
			log.Fatalf("failed to run gateway with DI: %v", err)
		}
	} else {
		log.Println("Starting with legacy architecture...")
		mainLegacy()
	}
}