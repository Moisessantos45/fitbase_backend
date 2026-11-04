package db

import (
	"fitbase/internal/models"
	"fmt"
	"log"
	"os"
)

func InitializeDatabase() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	prod := os.Getenv("GO_ENV")
	if prod == "prod" {
		fmt.Println("Running in production mode, skipping database migration")
		return nil
	}

	log.Println("Running in development mode, performing database migration")

	// Pre-sembrar tablas de lookup que son FK targets.
	// AutoMigrate agrega constraints y falla si los datos referenciados no existen.
	// if err := SeedLookupTables(); err != nil {
	// 	return fmt.Errorf("error pre-seeding lookup tables: %v", err)
	// }

	_ = DB.Exec("CREATE EXTENSION IF NOT EXISTS postgis")

	// Migrar el resto de los modelos (incluye los lookup que ya existen, se saltan)
	if err := DB.AutoMigrate(models.Models...); err != nil {
		return fmt.Errorf("error migrating database: %v", err)
	}

	if err := SeedDatabase(); err != nil {
		return fmt.Errorf("error seeding database: %v", err)
	}

	return nil
}
