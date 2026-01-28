package generator

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"project-scaffold/internal/templates"
)

type Stack string

const (
	StackGoGin       Stack = "Go (Gin)"
	StackNodeExpress Stack = "Node.js (Express)"
)

func ParseStackKey(key string) (Stack, error) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "go-gin":
		return StackGoGin, nil
	case "node-express":
		return StackNodeExpress, nil
	default:
		return "", fmt.Errorf("invalid stack %q (use: go-gin | node-express)", key)
	}
}

type Database string

const (
	DBPostgreSQL Database = "PostgreSQL"
	DBMongoDB   Database = "MongoDB"
	DBSQLite    Database = "SQLite"
)

func ParseDatabaseKey(key string) (Database, error) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "postgresql", "postgres":
		return DBPostgreSQL, nil
	case "mongodb", "mongo":
		return DBMongoDB, nil
	case "sqlite":
		return DBSQLite, nil
	default:
		return "", fmt.Errorf("invalid db %q (use: postgresql | mongodb | sqlite)", key)
	}
}

type Options struct {
	ProjectName string
	Stack       Stack
	Database    Database
	UseDocker   bool
}

type templateData struct {
	ProjectName string
	Stack       Stack
	Database    Database
	UseDocker   bool
}

func Generate(targetDir string, opts Options) error {
	if err := validate(opts); err != nil {
		return err
	}

	stackKey, err := stackDir(opts.Stack)
	if err != nil {
		return err
	}
	dbKey, err := dbDir(opts.Database)
	if err != nil {
		return err
	}

	base := filepath.ToSlash(filepath.Join("scaffolds", stackKey, dbKey))
	if _, err := fs.Stat(templates.FS, base); err != nil {
		return fmt.Errorf("template not found for stack=%q db=%q (expected %s): %w", opts.Stack, opts.Database, base, err)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	data := templateData{
		ProjectName: opts.ProjectName,
		Stack:       opts.Stack,
		Database:    opts.Database,
		UseDocker:   opts.UseDocker,
	}

	return fs.WalkDir(templates.FS, base, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, base+"/")
		if rel == path {
			rel = strings.TrimPrefix(path, base) // defensive
			rel = strings.TrimPrefix(rel, "/")
		}

		if !opts.UseDocker {
			if rel == "Dockerfile.tmpl" || rel == "docker-compose.yml.tmpl" {
				return nil
			}
		}

		dstRel := strings.TrimSuffix(rel, ".tmpl")
		dstRel = mapDotfiles(dstRel)
		dstPath := filepath.Join(targetDir, filepath.FromSlash(dstRel))

		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}

		b, err := fs.ReadFile(templates.FS, path)
		if err != nil {
			return err
		}

		tpl, err := template.New(rel).
			Option("missingkey=error").
			Parse(string(b))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", path, err)
		}

		var out bytes.Buffer
		if err := tpl.Execute(&out, data); err != nil {
			return fmt.Errorf("execute template %s: %w", path, err)
		}

		if err := os.WriteFile(dstPath, out.Bytes(), 0o644); err != nil {
			return err
		}
		return nil
	})
}

func mapDotfiles(p string) string {
	switch filepath.ToSlash(p) {
	case "env.example":
		return ".env.example"
	case "gitignore":
		return ".gitignore"
	default:
		return p
	}
}

func validate(opts Options) error {
	if strings.TrimSpace(opts.ProjectName) == "" {
		return errors.New("project name is required")
	}
	if _, err := stackDir(opts.Stack); err != nil {
		return err
	}
	if _, err := dbDir(opts.Database); err != nil {
		return err
	}
	return nil
}

func stackDir(s Stack) (string, error) {
	switch s {
	case StackGoGin:
		return "go-gin", nil
	case StackNodeExpress:
		return "node-express", nil
	default:
		return "", fmt.Errorf("unsupported stack: %q", s)
	}
}

func dbDir(d Database) (string, error) {
	switch d {
	case DBPostgreSQL:
		return "postgresql", nil
	case DBMongoDB:
		return "mongodb", nil
	case DBSQLite:
		return "sqlite", nil
	default:
		return "", fmt.Errorf("unsupported database: %q", d)
	}
}

