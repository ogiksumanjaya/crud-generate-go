package generator

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type Config struct {
	TargetProjectRoot string
	MigrationFile     string
	EntityDir         string
	RepositoryDir     string
	UsecaseDir        string
	HandlerDir        string
	SkipEntity        bool
	SkipRepository    bool
	SkipUsecase       bool
	SkipHandler       bool
}

type TemplateData struct {
	EntityName       string
	LowerEntityName  string
	TableName        string
	Fields           []FieldInfo
	SelectFields     string
	ScanFields       string
	InsertFields     string
	InsertValues     string
	UpdateFields     string
	CreateExecFields string
	UpdateExecFields string
	LastParamIndex   int
}

type FieldInfo struct {
	Name string
	Type string
	Tag  string
}

type SimpleGenerator struct {
	config    Config
	templates map[string]*template.Template
}

func NewGenerator(config Config) (*SimpleGenerator, error) {
	g := &SimpleGenerator{
		config:    config,
		templates: make(map[string]*template.Template),
	}

	if err := g.loadTemplates(); err != nil {
		return nil, err
	}

	return g, nil
}

func (g *SimpleGenerator) loadTemplates() error {
	files, err := templateFS.ReadDir("templates")
	if err != nil {
		return err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".tmpl") {
			continue
		}

		content, err := templateFS.ReadFile("templates/" + file.Name())
		if err != nil {
			return err
		}

		tmpl, err := template.New(file.Name()).Parse(string(content))
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", file.Name(), err)
		}

		g.templates[file.Name()] = tmpl
	}
	return nil
}

func (g *SimpleGenerator) Generate(tableName, entityName string) error {
	schema, err := g.readMigration()
	if err != nil {
		return fmt.Errorf("error reading migration: %w", err)
	}

	tableInfos, err := ParseSchema(schema, []string{tableName})
	if err != nil {
		return fmt.Errorf("error parsing schema: %w", err)
	}

	tableInfo, exists := tableInfos[tableName]
	if !exists {
		return fmt.Errorf("table %s not found in schema", tableName)
	}

	templateData := &TemplateData{
		EntityName:       entityName,
		LowerEntityName:  strings.ToLower(entityName),
		TableName:        tableName,
		Fields:           g.convertColumnsToFields(tableInfo.Columns),
		SelectFields:     g.generateSelectFields(tableInfo.Columns),
		ScanFields:       g.generateScanFields(tableInfo.Columns),
		InsertFields:     g.generateInsertFields(tableInfo.Columns),
		InsertValues:     g.generateInsertValues(tableInfo.Columns),
		UpdateFields:     g.generateUpdateFields(tableInfo.Columns),
		CreateExecFields: g.generateCreateExecFields(tableInfo.Columns),
		UpdateExecFields: g.generateUpdateExecFields(tableInfo.Columns),
		LastParamIndex:   len(tableInfo.Columns),
	}

	if !g.config.SkipEntity {
		if err := g.generateEntity(entityName, templateData); err != nil {
			return err
		}
	}

	if !g.config.SkipRepository {
		if err := g.generateRepository(entityName, templateData); err != nil {
			return err
		}
	}

	if !g.config.SkipUsecase {
		if err := g.generateUsecase(entityName, templateData); err != nil {
			return err
		}
	}

	if !g.config.SkipHandler {
		if err := g.generateHandler(entityName, templateData); err != nil {
			return err
		}
	}

	return nil
}

func (g *SimpleGenerator) convertColumnsToFields(columns []ColumnInfo) []FieldInfo {
	fields := make([]FieldInfo, len(columns))
	for i, col := range columns {
		fields[i] = FieldInfo{
			Name: g.toGoFieldName(col.Name),
			Type: g.toGoType(col.Type, col.Nullable),
			Tag:  col.Name,
		}
	}
	return fields
}

func (g *SimpleGenerator) toGoFieldName(dbName string) string {
	words := strings.Split(dbName, "_")
	var result string
	caser := cases.Title(language.English)
	for _, word := range words {
		if word == "id" {
			result += "ID"
		} else {
			result += caser.String(word)
		}
	}
	return result
}

func (g *SimpleGenerator) toGoType(dbType string, nullable bool) string {
	baseType := strings.ToLower(dbType)

	// Mapping tipe data SQL ke Go
	var goType string
	switch {
	case strings.Contains(baseType, "int"):
		if strings.Contains(baseType, "big") {
			goType = "int64"
		} else {
			goType = "int"
		}
	case strings.Contains(baseType, "varchar"),
		strings.Contains(baseType, "text"),
		strings.Contains(baseType, "char"):
		goType = "string"
	case strings.Contains(baseType, "bool"):
		goType = "bool"
	case strings.Contains(baseType, "timestamp"),
		strings.Contains(baseType, "date"):
		goType = "time.Time"
	case strings.Contains(baseType, "numeric"),
		strings.Contains(baseType, "decimal"):
		goType = "float64"
	default:
		goType = "string"
	}

	// Jika nullable, gunakan pointer
	if nullable && goType != "string" {
		return "*" + goType
	}

	return goType
}

func (g *SimpleGenerator) generateSelectFields(columns []ColumnInfo) string {
	fields := make([]string, len(columns))
	for i, col := range columns {
		fields[i] = col.Name
	}
	return strings.Join(fields, ", ")
}

func (g *SimpleGenerator) generateScanFields(columns []ColumnInfo) string {
	fields := make([]string, len(columns))
	for i := range columns {
		fields[i] = fmt.Sprintf("&data.%s", g.toGoFieldName(columns[i].Name))
	}
	return strings.Join(fields, ", ")
}

func (g *SimpleGenerator) generateInsertFields(columns []ColumnInfo) string {
	fields := make([]string, len(columns))
	for i, col := range columns {
		fields[i] = col.Name
	}
	return strings.Join(fields, ", ")
}

func (g *SimpleGenerator) generateInsertValues(columns []ColumnInfo) string {
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(placeholders, ", ")
}

func (g *SimpleGenerator) generateUpdateFields(columns []ColumnInfo) string {
	fields := make([]string, len(columns))
	for i, col := range columns {
		fields[i] = fmt.Sprintf("%s = $%d", col.Name, i+1)
	}
	return strings.Join(fields, ", ")
}

func (g *SimpleGenerator) generateCreateExecFields(columns []ColumnInfo) string {
	fields := make([]string, len(columns))
	for i := range columns {
		fields[i] = fmt.Sprintf("data.%s", g.toGoFieldName(columns[i].Name))
	}
	return strings.Join(fields, ", ")
}

func (g *SimpleGenerator) generateUpdateExecFields(columns []ColumnInfo) string {
	fields := make([]string, len(columns))
	for i := range columns {
		fields[i] = fmt.Sprintf("data.%s", g.toGoFieldName(columns[i].Name))
	}
	return strings.Join(fields, ", ")
}

func (g *SimpleGenerator) readMigration() (string, error) {
	if _, err := os.Stat(g.config.MigrationFile); os.IsNotExist(err) {
		return "", fmt.Errorf("migration file not found: %s", g.config.MigrationFile)
	}

	content, err := os.ReadFile(g.config.MigrationFile)
	if err != nil {
		return "", fmt.Errorf("error reading migration file: %w", err)
	}

	return string(content), nil
}

func (g *SimpleGenerator) generateEntity(entityName string, data *TemplateData) error {
	entityFile := fmt.Sprintf("%s/%s.go", g.config.EntityDir, strings.ToLower(entityName))

	f, err := os.Create(entityFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return g.templates["entity.tmpl"].Execute(f, data)
}

func (g *SimpleGenerator) generateRepository(entityName string, data *TemplateData) error {
	repoDir := fmt.Sprintf("%s/%s", g.config.RepositoryDir, strings.ToLower(entityName))
	if err := os.MkdirAll(repoDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating repository directory: %w", err)
	}

	repoFile := fmt.Sprintf("%s/postgres.go", repoDir)

	f, err := os.Create(repoFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return g.templates["repository.tmpl"].Execute(f, data)
}

func (g *SimpleGenerator) generateUsecase(entityName string, data *TemplateData) error {
	usecaseFile := fmt.Sprintf("%s/%s_usecase.go", g.config.UsecaseDir, strings.ToLower(entityName))

	f, err := os.Create(usecaseFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return g.templates["usecase.tmpl"].Execute(f, data)
}

func (g *SimpleGenerator) generateHandler(entityName string, data *TemplateData) error {
	handlerFile := fmt.Sprintf("%s/%s.go", g.config.HandlerDir, strings.ToLower(entityName))

	f, err := os.Create(handlerFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return g.templates["handler.tmpl"].Execute(f, data)
}

type TableInfo struct {
	Columns []ColumnInfo
}

type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
}

func ParseSchema(schema string, targetTables []string) (map[string]*TableInfo, error) {
	tableInfos := make(map[string]*TableInfo)
	var currentTable string
	var inTargetTable bool

	lines := strings.Split(schema, "\n")
	targetTableSet := make(map[string]bool)
	for _, table := range targetTables {
		targetTableSet[table] = true
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		if strings.HasPrefix(line, "CREATE TABLE") {
			inTargetTable = false
			for table := range targetTableSet {
				if strings.Contains(line, table) {
					inTargetTable = true
					currentTable = table
					tableInfos[currentTable] = &TableInfo{}
					break
				}
			}
			continue
		}

		if inTargetTable && strings.Contains(line, " ") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			columnName := strings.Trim(parts[0], `",`)
			columnType := strings.SplitN(parts[1], "(", 2)[0]

			nullable := !strings.Contains(strings.ToUpper(line), "NOT NULL")

			column := ColumnInfo{
				Name:     columnName,
				Type:     columnType,
				Nullable: nullable,
			}

			tableInfos[currentTable].Columns = append(tableInfos[currentTable].Columns, column)
		}

		if inTargetTable && strings.HasSuffix(line, ");") {
			inTargetTable = false
		}
	}

	if len(tableInfos) == 0 {
		return nil, fmt.Errorf("none of the specified tables found in schema")
	}

	return tableInfos, nil
}
