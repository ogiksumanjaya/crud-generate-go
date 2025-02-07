package generator

import (
	"context"
	"embed"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

//go:embed templates/*.txt
var templateFS embed.FS

type Config struct {
	TargetProjectRoot string
	MigrationFile     string
	EntityDir         string
	RepositoryDir     string
	APITableName      string
}

type SimpleGenerator struct {
	model    *genai.GenerativeModel
	config   Config
	examples map[string]string
}

func NewGenerator(apiKey string, config Config) (*SimpleGenerator, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	g := &SimpleGenerator{
		model:    client.GenerativeModel("gemini-1.5-flash"),
		config:   config,
		examples: make(map[string]string),
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
		content, err := templateFS.ReadFile("templates/" + file.Name())
		if err != nil {
			return err
		}
		g.examples[file.Name()] = string(content)
	}
	return nil
}

func (g *SimpleGenerator) Generate(tableName string) error {
	migrationContent, err := g.readMigration()
	if err != nil {
		return fmt.Errorf("error reading migration: %w", err)
	}

	prompt := g.buildPrompt(tableName, migrationContent)

	ctx := context.Background()
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return err
	}

	return g.saveGeneratedCode(tableName, resp.Candidates[0].Content.Parts[0].(genai.Text))
}

func (g *SimpleGenerator) readMigration() (string, error) {
	if _, err := os.Stat(g.config.MigrationFile); os.IsNotExist(err) {
		return "", fmt.Errorf("migration file not found: %s", g.config.MigrationFile)
	}

	content, err := os.ReadFile(g.config.MigrationFile)
	if err != nil {
		return "", fmt.Errorf("error reading migration file: %w", err)
	}

	if !strings.Contains(string(content), "CREATE TABLE") {
		return "", fmt.Errorf("invalid migration file format")
	}

	return string(content), nil
}

func (g *SimpleGenerator) buildPrompt(tableName, migrationContent string) string {
	return fmt.Sprintf(`
        ANDA ADALAH GENERATOR KODE GO YANG HARUS MENGIKUTI TEMPLATE DENGAN KETAT. 
        TOLONG IKUTI SEMUA ATURAN DI BAWAH INI:

        **PERATURAN KETAT:**
        1. HANYA gunakan template yang disediakan. JANGAN menambahkan fitur, method, atau logika baru.
        2. JANGAN mengubah struktur template. Ikuti persis seperti contoh.
        3. JANGAN menambahkan komentar dalam bentuk apapun, kecuali yang ada di "PROMPT NOTE".
        4. PASTIKAN tidak ada satupun karakter komentar (// atau /* */) kecuali yang disebutkan di "PROMPT NOTE".
        5. JIKA ada komentar "// PROMPT NOTE:" atau "-- PROMPT NOTE:", ikuti instruksi di dalamnya sebagai prompt tambahan.
        6. SESUAIKAN dengan skema database yang diberikan. Gunakan nama kolom dan tipe data yang sesuai.
        7. JIKA nama tabel berisi quote (contoh: "user"), berarti itu reserved keyword. Rubah nama tabel menjadi huruf kecil semua (contoh: user).
        8. FORMAT OUTPUT HARUS PERSIS SEPERTI CONTOH. JANGAN menambahkan penjelasan, komentar, atau formatting tambahan.

        **CONTOH ENTITY:**
        %s

        **CONTOH REPOSITORY:**
        %s

        **SKEMA DATABASE:**
        %s

        **TUGAS:**
        Buatkan 2 file untuk tabel "%s":
        - File entity dengan format [nama_tabel].go
        - File repository dengan format [nama_tabel]_repository.go

        **HANDLE PROMPT NOTE:**
        - Jika ada komentar "// PROMPT NOTE:" atau "-- PROMPT NOTE:", ikuti instruksi di dalamnya.
        - Contoh: Jika ada "-- PROMPT NOTE: Filter Search: email, status", tambahkan fitur filter search di repository untuk kolom email dan status.
        - Contoh: Jika ada "-- PROMPT NOTE: Filter by Status with Multiple value", tambahkan fitur filter status dengan multiple value di repository.

        **FORMAT OUTPUT:**
        [ENTITY]
        <kode entity>
        
        [REPOSITORY]
        <kode repository>

        **CATATAN:**
        - JANGAN menambahkan penjelasan, komentar, atau formatting tambahan.
        - HANYA kembalikan kode yang sudah disesuaikan dengan format di atas.
    `,
		g.examples["user_entity.go"],
		g.examples["user_repository.go"],
		migrationContent,
		tableName,
	)
}

func (g *SimpleGenerator) validateAndCleanCode(code string) (string, string, error) {
	// Bersihkan markdown
	code = strings.ReplaceAll(code, "```go", "")
	code = strings.Trim(code, "` \n")

	// Hapus semua komentar
	code = regexp.MustCompile(`(?s)//.*?\n|/\*.*?\*/`).ReplaceAllString(code, "")

	// Split entity dan repository
	parts := strings.Split(code, "[ENTITY]")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("format entity tidak valid")
	}

	entityRepo := strings.Split(parts[1], "[REPOSITORY]")
	if len(entityRepo) != 2 {
		return "", "", fmt.Errorf("format repository tidak valid")
	}

	entityCode := strings.TrimSpace(entityRepo[0])
	repoCode := strings.TrimSpace(entityRepo[1])

	// Validasi dasar
	if strings.Contains(entityCode, "//") || strings.Contains(repoCode, "//") {
		return "", "", fmt.Errorf("masih terdapat komentar dalam kode")
	}

	return entityCode, repoCode, nil
}

func (g *SimpleGenerator) saveGeneratedCode(tableName string, content genai.Text) error {
	entityCode, repoCode, err := g.validateAndCleanCode(string(content))
	if err != nil {
		return fmt.Errorf("validasi kode gagal: %w", err)
	}

	entityFile := fmt.Sprintf("%s/%s.go", g.config.EntityDir, strings.ToLower(tableName))
	if err := os.WriteFile(entityFile, []byte(entityCode), 0644); err != nil {
		return err
	}

	repoFolder := fmt.Sprintf("%s/%s", g.config.RepositoryDir, strings.ToLower(tableName))
	if err := os.MkdirAll(repoFolder, 0755); err != nil {
		return fmt.Errorf("gagal membuat folder repository untuk tabel %s: %w", tableName, err)
	}

	repoFile := fmt.Sprintf("%s/%s/postgres.go", g.config.RepositoryDir, strings.ToLower(tableName))
	return os.WriteFile(repoFile, []byte(repoCode), 0644)
}
