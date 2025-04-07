# CRUD Generator

A simple and flexible CRUD (Create, Read, Update, Delete) code generator for Go projects.

## Overview

This CRUD Generator automatically generates boilerplate code for:

- Entity structs
- Repository layer with PostgreSQL implementation
- Use case/business logic layer
- HTTP handlers with Swagger documentation

## Features

- Generates complete CRUD operations from database schema
- Supports PostgreSQL data types
- Generates Swagger/OpenAPI documentation
- Follows clean architecture principles
- Fully customizable external templates
- Handles nullable fields appropriately

## How to use

1. Build project

```bash
go build -o crud-generator ./cmd/crud-generator/main.go
```

2. Run with this command:

```bash
./crud-generator --migration-file=migrations/postgres/000001_initial_db_migration.up.sql --table="users" --template-dir="./templates"
```

3. Command with skip flag:

```bash
./crud-generator --migration-file=migrations/postgres/000001_initial_db_migration.up.sql --table="users" --template-dir="./templates" --skip-entity --skip-repository
```

## Template System

Templates are now fully external and must be provided through the `--template-dir` flag.

### Template Directives

Each template file can include special directives at the top of the file to specify where the generated file should be placed:

```
#path = /path/to/output/directory
#fileName: = filename_template.go
```

For example:

```
#path = /repository
#fileName: = /{{.TableName}}/postgres.go

package {{.TableName}}

// Template content...
```

Both directives support Go template syntax using the same variables available in the template content.

If directives are not provided, the generator uses fallback paths based on the template filename.

### Creating Custom Templates

1. Create a templates directory:

```bash
mkdir -p templates
```

2. Create your template files with .tmpl extension and proper directives:

- `entity.tmpl` - Entity structures
- `repository.tmpl` - Repository implementation
- `usecase.tmpl` - Business logic
- `handler.tmpl` - HTTP handlers
- `payload.tmpl` - Request/response payloads

## Template Variables

When customizing templates, you can use the following variables:

| Variable               | Description                    | Example             |
| ---------------------- | ------------------------------ | ------------------- |
| `{{.EntityName}}`      | Capitalized entity name        | "User", "Product"   |
| `{{.LowerEntityName}}` | Lowercase entity name          | "user", "product"   |
| `{{.TableName}}`       | Database table name            | "users", "products" |
| `{{.LastParamIndex}}`  | Last parameter index (integer) | 5                   |

### SQL Generation Variables

| Variable                | Description                       | Example                                              |
| ----------------------- | --------------------------------- | ---------------------------------------------------- |
| `{{.SelectFields}}`     | SQL fields for SELECT queries     | "id, name, email, created_at"                        |
| `{{.ScanFields}}`       | Fields for scanning database rows | "&data.ID, &data.Name, &data.Email, &data.CreatedAt" |
| `{{.InsertFields}}`     | Fields for INSERT queries         | "id, name, email, created_at"                        |
| `{{.InsertValues}}`     | Placeholders for INSERT queries   | "$1, $2, $3, $4"                                     |
| `{{.UpdateFields}}`     | SET clauses for UPDATE queries    | "name = $1, email = $2, updated_at = $3"             |
| `{{.CreateExecFields}}` | Fields for INSERT execution       | "data.ID, data.Name, data.Email, data.CreatedAt"     |
| `{{.UpdateExecFields}}` | Fields for UPDATE execution       | "data.Name, data.Email, data.UpdatedAt, data.ID"     |

### Fields Collection

The `{{.Fields}}` variable provides an array of field information. Each field has the following properties:

```go
{{range .Fields}}
  {{.Name}} // Field name in Go format (e.g., "UserID")
  {{.Type}} // Go type (e.g., "string", "int", "*time.Time")
  {{.Tag}}  // Database column name (e.g., "user_id")
{{end}}
```

Example usage in a template:

```go
type {{.EntityName}}Ent struct {
  {{range .Fields}}
  {{.Name}} {{.Type}} `db:"{{.Tag}}"`
  {{end}}
}
```

## Example Template with Directives

```go
#path = /core/entity
#fileName: = {{.LowerEntityName}}.go

package entity

import "time"

type {{.EntityName}}Ent struct {
  {{range .Fields}}
  {{.Name}} {{.Type}} `db:"{{.Tag}}"`
  {{end}}
}
```
