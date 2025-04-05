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
- Customizable templates
- Handles nullable fields appropriately

## How to use

1. Build project

```bash
go build -o crud-generator ./cmd/crud-generator/main.go
```

2. Run with this command:

```bash
./crud-generator  --migration-file=migrations/postgres/000001_initial_db_migration.up.sql  --table="users"
```

3. Command with skip flag:

```bash
./crud-generator  --migration-file=migrations/postgres/000001_initial_db_migration.up.sql  --table="users" --skip-entity --skip-repository --skip-usecase --skip-handler
```
