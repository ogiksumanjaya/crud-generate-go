package main

import (
	"flag"
	"log"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ogiksumanjaya/crud-generator/generator"
)

func main() {
	config := generator.Config{
		TargetProjectRoot: ".",           // hardcoded ke directory saat ini
		EntityDir:         "core/entity", // hardcoded ke folder entity
		RepositoryDir:     "repository",  // hardcoded ke folder repository
		UsecaseDir:        "core/module", // hardcoded ke folder usecase
		HandlerDir:        "handler/api", // hardcoded ke folder api
	}

	// Menerima parameter tabel yang dipisahkan koma
	flag.StringVar(&config.MigrationFile, "migration-file", "", "Path to specific migration file (required)")
	tables := flag.String("table", "", "Comma-separated list of database table names to generate (required)")

	flag.Parse()

	if config.MigrationFile == "" {
		log.Fatal("Migration file is required. Use --migration-file flag")
	}

	if *tables == "" {
		log.Fatal("Table names are required. Use --table flag")
	}

	g, err := generator.NewGenerator(config)
	if err != nil {
		log.Fatal(err)
	}

	// Memproses setiap tabel
	tableList := strings.Split(*tables, ",")
	for _, tableName := range tableList {
		tableName = strings.TrimSpace(tableName)
		caser := cases.Title(language.English)
		finalEntityName := strings.Replace(tableName, "_", " ", -1)
		words := strings.Fields(finalEntityName)
		for i, word := range words {
			words[i] = caser.String(word)
		}
		finalEntityName = strings.Join(words, "")

		if err := g.Generate(tableName, finalEntityName); err != nil {
			log.Fatalf("Error generating for table %s: %v", tableName, err)
		}

		log.Printf("CRUD generated successfully for entity: %s!", finalEntityName)
	}
}
