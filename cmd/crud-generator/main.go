package main

import (
	"flag"
	"log"
	"os"

	"github.com/ogiksumanjaya/crud-generator/generator"
)

func main() {
	config := generator.Config{}

	flag.StringVar(&config.TargetProjectRoot, "target", ".", "Root directory of target project")
	flag.StringVar(&config.MigrationFile, "migration-file", "", "Path to specific migration file (required)")
	flag.StringVar(&config.EntityDir, "entities", "core/entity", "Entity output directory")
	flag.StringVar(&config.RepositoryDir, "repositories", "repository", "Repository output directory")
	flag.StringVar(&config.APITableName, "table", "", "Database table name to generate (required)")

	apiKey := flag.String("api-key", os.Getenv("GEMINI_API_KEY"), "Gemini API key")

	flag.Parse()

	if *apiKey == "" {
		log.Fatal("Missing Gemini API key. Set GEMINI_API_KEY environment variable or use --api-key flag")
	}

	if config.MigrationFile == "" {
		log.Fatal("Migration file is required. Use --migration-file flag")
	}

	if config.APITableName == "" {
		log.Fatal("Table name is required. Use --table flag")
	}

	g, err := generator.NewGenerator(*apiKey, config)
	if err != nil {
		log.Fatal(err)
	}

	if err := g.Generate(config.APITableName); err != nil {
		log.Fatal(err)
	}

	log.Println("CRUD generated successfully!")
}
