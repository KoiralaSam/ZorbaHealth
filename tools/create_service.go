package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	serviceName := flag.String("name", "", "Name of the service (e.g., user, payment)")
	flag.Parse()

	if *serviceName == "" {
		fmt.Println("Please provide a service name using -name flag")
		os.Exit(1)
	}

	// Create service directory structure (matches patient-service layout)
	basePath := filepath.Join("services", *serviceName+"-service")
	dirs := []string{
		filepath.Join("cmd", *serviceName+"-service"),
		"config",
		"internal/adapters/primary/grpc/handlers",
		"internal/adapters/primary/grpc/mappers",
		"internal/adapters/primary/http/handlers",
		"internal/adapters/primary/http/middleware",
		"internal/adapters/secondary/repositories/postgres/query",
		"internal/adapters/secondary/repositories/postgres/sqlc",
		"internal/core/domain/models",
		"internal/core/ports/outbound",
		"internal/core/services",
		"migrations",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(basePath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Successfully created %s service folder structure in %s\n", *serviceName, basePath)
	fmt.Println("\nDirectory structure created:")
	fmt.Printf(`
services/%s-service/
├── cmd/%s-service/
├── config/
├── internal/
│   ├── adapters/
│   │   ├── primary/
│   │   │   ├── grpc/
│   │   │   │   ├── handlers/
│   │   │   │   └── mappers/
│   │   │   └── http/
│   │   │       ├── handlers/
│   │   │       └── middleware/
│   │   └── secondary/
│   │       └── repositories/
│   │           └── postgres/
│   │               ├── query/
│   │               └── sqlc/
│   └── core/
│       ├── domain/
│       │   └── models/
│       ├── ports/
│       │   └── outbound/
│       └── services/
└── migrations/
`, *serviceName, *serviceName)
}
