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
		EntityDir:         "core/entity", // fallback directory
		RepositoryDir:     "repository",  // fallback directory
		UsecaseDir:        "core/module", // fallback directory
		HandlerDir:        "handler/api", // fallback directory
		PayloadDir:        "payload",     // fallback directory
	}

	// Menerima parameter tabel yang dipisahkan koma
	flag.StringVar(&config.MigrationFile, "migration-file", "", "Path to specific migration file (required)")
	tables := flag.String("table", "", "Comma-separated list of database table names to generate (required)")

	// Template directory is now required
	flag.StringVar(&config.TemplateDir, "template-dir", "", "Path to templates directory (required)")

	// Add skip flags
	flag.BoolVar(&config.SkipEntity, "skip-entity", false, "Skip entity generation")
	flag.BoolVar(&config.SkipRepository, "skip-repository", false, "Skip repository generation")
	flag.BoolVar(&config.SkipUsecase, "skip-usecase", false, "Skip usecase generation")
	flag.BoolVar(&config.SkipHandler, "skip-handler", false, "Skip handler generation")
	flag.BoolVar(&config.SkipPayload, "skip-payload", false, "Skip payload generation")

	flag.Parse()

	if config.MigrationFile == "" {
		log.Fatal("Migration file is required. Use --migration-file flag")
	}

	if *tables == "" {
		log.Fatal("Table names are required. Use --table flag")
	}

	if config.TemplateDir == "" {
		log.Fatal("Template directory is required. Use --template-dir flag")
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
