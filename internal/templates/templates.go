package templates

import "embed"

// FS contains all scaffold templates embedded into the binary.
//
// Layout:
//   scaffolds/<stack>/<database>/<files...>.tmpl
//
// Example:
//   scaffolds/go-gin/postgresql/cmd/main.go.tmpl
//
// Embed the scaffolds directory recursively.
// Note: go:embed does not include dotfiles (so dotfiles are handled via non-dot template names).
//go:embed scaffolds
var FS embed.FS

