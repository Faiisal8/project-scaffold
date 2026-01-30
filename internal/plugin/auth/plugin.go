package auth

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"project-scaffold/internal/plugin"
)

const marker = "// scaffold:auth"

//go:embed templates
var templatesFS embed.FS

type authPlugin struct{}

func init() {
	plugin.Register(&authPlugin{})
}

func (*authPlugin) Name() string {
	return "auth"
}

func (*authPlugin) CompatibleStacks() []string {
	return []string{"go-gin", "node-express", "node-express-ts"}
}

func (p *authPlugin) Apply(ctx *plugin.Context) error {
	switch ctx.StackKey {
	case "go-gin":
		return p.applyGoGin(ctx)
	case "node-express":
		return p.applyNodeExpress(ctx)
	case "node-express-ts":
		return p.applyNodeExpressTS(ctx)
	default:
		return fmt.Errorf("auth plugin: unsupported stack %q", ctx.StackKey)
	}
}

func (p *authPlugin) applyGoGin(ctx *plugin.Context) error {
	data := map[string]string{"ProjectName": ctx.ProjectName}
	if err := p.writeTemplates(ctx.TargetDir, "go-gin", data); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	injection := "\tauthHandler := handlers.NewAuthHandler()\n\troutes.RegisterAuth(router, authHandler)\n"
	if err := injectAtMarker(filepath.Join(ctx.TargetDir, "cmd", "main.go"), marker, injection); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	return p.appendEnvExample(ctx.TargetDir, "JWT_SECRET=change-me\n")
}

func (p *authPlugin) applyNodeExpress(ctx *plugin.Context) error {
	data := map[string]string{"ProjectName": ctx.ProjectName}
	if err := p.writeTemplates(ctx.TargetDir, "node-express", data); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	if err := injectAtMarker(filepath.Join(ctx.TargetDir, "src", "server.js"), "// scaffold:auth-import", "import authRouter from \"./routes/auth.js\";"); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	if err := injectAtMarker(filepath.Join(ctx.TargetDir, "src", "server.js"), "// scaffold:auth-routes", "app.use(\"/auth\", authRouter);"); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	return p.appendEnvExample(ctx.TargetDir, "JWT_SECRET=change-me\n")
}

func (p *authPlugin) applyNodeExpressTS(ctx *plugin.Context) error {
	data := map[string]string{"ProjectName": ctx.ProjectName}
	if err := p.writeTemplates(ctx.TargetDir, "node-express-ts", data); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	if err := injectAtMarker(filepath.Join(ctx.TargetDir, "src", "server.ts"), "// scaffold:auth-import", "import authRouter from \"./routes/auth.js\";"); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	if err := injectAtMarker(filepath.Join(ctx.TargetDir, "src", "server.ts"), "// scaffold:auth-routes", "app.use(\"/auth\", authRouter);"); err != nil {
		return fmt.Errorf("auth plugin: %w", err)
	}
	return p.appendEnvExample(ctx.TargetDir, "JWT_SECRET=change-me\n")
}

func (p *authPlugin) writeTemplates(targetDir, stackKey string, data map[string]string) error {
	base := filepath.ToSlash(filepath.Join("templates", stackKey))
	return fs.WalkDir(templatesFS, base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}
		content, err := fs.ReadFile(templatesFS, path)
		if err != nil {
			return err
		}
		tpl, err := template.New(path).Parse(string(content))
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, data); err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, base+"/")
		rel = strings.TrimSuffix(rel, ".tmpl")
		dstPath := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dstPath, buf.Bytes(), 0o644)
	})
}

func injectAtMarker(filePath, markerLine, injection string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var found bool
	for i, line := range lines {
		if strings.TrimSpace(line) != strings.TrimSpace(markerLine) {
			continue
		}
		found = true
		indent := ""
		for _, c := range line {
			if c == ' ' || c == '\t' {
				indent += string(c)
			} else {
				break
			}
		}
		injectLines := strings.Split(strings.TrimSuffix(injection, "\n"), "\n")
		var newLines []string
		for _, inj := range injectLines {
			if inj != "" {
				newLines = append(newLines, indent+inj)
			}
		}
		rest := append([]string{line}, newLines...)
		lines = append(lines[:i], append(rest, lines[i+1:]...)...)
		break
	}
	if !found {
		return fmt.Errorf("required marker %q not found in %s", markerLine, filePath)
	}
	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o644)
}

func (p *authPlugin) appendEnvExample(targetDir, line string) error {
	path := filepath.Join(targetDir, ".env.example")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(string(b), "\n") {
		line = "\n" + line
	}
	return os.WriteFile(path, append(b, line...), 0o644)
}
