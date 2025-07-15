package main

import (
	"log"

	"github.com/jamesprial/nexus/internal/gateway"
)

func mainLegacy() {
	if err := gateway.Run(); err != nil {
		log.Fatalf("failed to run gateway: %v", err)
	}
}
