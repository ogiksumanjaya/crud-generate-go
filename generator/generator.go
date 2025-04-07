package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Removed embedded templates FS

type Config struct {
	TargetProjectRoot string
	MigrationFile     string
	EntityDir         string // Now used as fallback if not specified in template
	RepositoryDir     string // Now used as fallback if not specified in template
	UsecaseDir        string // Now used as fallback if not specified in template
	HandlerDir        string // Now used as fallback if not specified in template
	PayloadDir        string // Now used as fallback if not specified in template
	TemplateDir       string // Required - directory containing templates
	SkipEntity        bool
	SkipRepository    bool
	SkipUsecase       bool
	SkipHandler       bool
	SkipPayload       bool
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

type TemplateInfo struct {
	Name     string
	Content  string
	Path     string // Output directory path from template comment
	FileName string // Output file name from template comment
}

type FieldInfo struct {
	Name string
	Type string
	Tag  string
}

type SimpleGenerator struct {
	config       Config
	templates    map[string]*template.Template
	templateInfo map[string]TemplateInfo
}

func NewGenerator(config Config) (*SimpleGenerator, error) {
	g := &SimpleGenerator{
		config:       config,
		templates:    make(map[string]*template.Template),
		templateInfo: make(map[string]TemplateInfo),
	}

	if err := g.loadTemplates(); err != nil {
		return nil, err
	}

	return g, nil
}

func (g *SimpleGenerator) loadTemplates() error {
	// Check if template directory exists
	if g.config.TemplateDir == "" {
		return fmt.Errorf("template directory is required")
	}

	if _, err := os.Stat(g.config.TemplateDir); os.IsNotExist(err) {
		return fmt.Errorf("template directory does not exist: %s", g.config.TemplateDir)
	}

	// List all .tmpl files in the directory
	files, err := os.ReadDir(g.config.TemplateDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no templates found in directory: %s", g.config.TemplateDir)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".tmpl") {
			continue
		}

		fullPath := filepath.Join(g.config.TemplateDir, file.Name())
		info, err := g.parseTemplateFile(fullPath, file.Name())
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", fullPath, err)
		}

		g.templateInfo[file.Name()] = info

		// Parse template
		tmpl, err := template.New(file.Name()).Parse(info.Content)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %w", file.Name(), err)
		}

		g.templates[file.Name()] = tmpl
	}

	// Check if we have some templates
	if len(g.templates) == 0 {
		return fmt.Errorf("no valid templates found in directory: %s", g.config.TemplateDir)
	}

	return nil
}

// parseTemplateFile reads a template file and extracts path and filename directives
func (g *SimpleGenerator) parseTemplateFile(filePath string, fileName string) (TemplateInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return TemplateInfo{}, err
	}
	defer file.Close()

	info := TemplateInfo{
		Name: fileName,
	}

	// Read the file by lines to extract directives
	scanner := bufio.NewScanner(file)
	var contentBuilder strings.Builder
	var readingDirectives = true

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if readingDirectives {
			if strings.HasPrefix(trimmed, "#path =") {
				info.Path = strings.TrimSpace(strings.TrimPrefix(trimmed, "#path ="))
				continue
			} else if strings.HasPrefix(trimmed, "#fileName:") || strings.HasPrefix(trimmed, "#fileName =") {
				info.FileName = strings.TrimSpace(strings.SplitN(trimmed, "=", 2)[1])
				continue
			} else if trimmed == "" {
				// Skip empty lines in directives section
				continue
			} else {
				// First non-directive line, end directive parsing
				readingDirectives = false
				contentBuilder.WriteString(line + "\n")
			}
		} else {
			contentBuilder.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return TemplateInfo{}, err
	}

	// Set content from builder
	info.Content = contentBuilder.String()

	return info, nil
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

	// Process each template
	for name, tmpl := range g.templates {
		info := g.templateInfo[name]

		// Skip based on name convention (fallback)
		if strings.HasPrefix(name, "entity") && g.config.SkipEntity {
			continue
		}
		if strings.HasPrefix(name, "repository") && g.config.SkipRepository {
			continue
		}
		if strings.HasPrefix(name, "usecase") && g.config.SkipUsecase {
			continue
		}
		if strings.HasPrefix(name, "handler") && g.config.SkipHandler {
			continue
		}
		if strings.HasPrefix(name, "payload") && g.config.SkipPayload {
			continue
		}

		// Generate output file path
		err := g.generateFromTemplate(name, tmpl, info, templateData)
		if err != nil {
			return fmt.Errorf("error generating from template %s: %w", name, err)
		}
	}

	return nil
}

func (g *SimpleGenerator) generateFromTemplate(templateName string, tmpl *template.Template, info TemplateInfo, data *TemplateData) error {
	// Process path and filename templates
	var pathBuf, fileNameBuf strings.Builder

	if info.Path != "" {
		pathTemplate, err := template.New("path").Parse(info.Path)
		if err != nil {
			return fmt.Errorf("invalid path template: %w", err)
		}
		if err := pathTemplate.Execute(&pathBuf, data); err != nil {
			return fmt.Errorf("error executing path template: %w", err)
		}
	}

	if info.FileName != "" {
		fileNameTemplate, err := template.New("fileName").Parse(info.FileName)
		if err != nil {
			return fmt.Errorf("invalid fileName template: %w", err)
		}
		if err := fileNameTemplate.Execute(&fileNameBuf, data); err != nil {
			return fmt.Errorf("error executing fileName template: %w", err)
		}
	}

	// Determine output path and filename
	var outputPath, outputFileName string

	// Use template path directives or fall back to config
	if pathBuf.String() != "" {
		outputPath = filepath.Join(g.config.TargetProjectRoot, strings.TrimPrefix(pathBuf.String(), "/"))
	} else {
		// Fallback based on template name
		if strings.HasPrefix(templateName, "entity") {
			outputPath = g.config.EntityDir
		} else if strings.HasPrefix(templateName, "repository") {
			outputPath = g.config.RepositoryDir
		} else if strings.HasPrefix(templateName, "usecase") {
			outputPath = g.config.UsecaseDir
		} else if strings.HasPrefix(templateName, "handler") {
			outputPath = g.config.HandlerDir
		} else if strings.HasPrefix(templateName, "payload") {
			outputPath = g.config.PayloadDir
		} else {
			outputPath = g.config.TargetProjectRoot
		}
	}

	// Use template fileName directive or fallback
	if fileNameBuf.String() != "" {
		if strings.HasPrefix(fileNameBuf.String(), "/") {
			parts := strings.Split(strings.TrimPrefix(fileNameBuf.String(), "/"), "/")

			// If multiple parts, the first n-1 parts are subdirectories
			if len(parts) > 1 {
				subdirs := parts[:len(parts)-1]
				outputPath = filepath.Join(outputPath, filepath.Join(subdirs...))
				outputFileName = parts[len(parts)-1]
			} else {
				outputFileName = parts[0]
			}
		} else {
			outputFileName = fileNameBuf.String()
		}
	} else {
		// Fallback names
		lowerName := strings.ToLower(data.EntityName)
		if strings.HasPrefix(templateName, "entity") {
			outputFileName = fmt.Sprintf("%s.go", lowerName)
		} else if strings.HasPrefix(templateName, "repository") {
			outputPath = filepath.Join(outputPath, lowerName)
			outputFileName = "postgres.go"
		} else if strings.HasPrefix(templateName, "usecase") {
			outputFileName = fmt.Sprintf("%s_usecase.go", lowerName)
		} else if strings.HasPrefix(templateName, "handler") {
			outputFileName = fmt.Sprintf("%s.go", lowerName)
		} else if strings.HasPrefix(templateName, "payload") {
			outputFileName = fmt.Sprintf("%s.go", lowerName)
		} else {
			outputFileName = fmt.Sprintf("%s.go", templateName[:len(templateName)-5]) // remove .tmpl
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", outputPath, err)
	}

	fullPath := filepath.Join(outputPath, outputFileName)
	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", fullPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	fmt.Printf("Generated: %s\n", fullPath)
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
