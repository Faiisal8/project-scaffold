package main

import (
	"project-scaffold/internal/cli"
	_ "project-scaffold/internal/plugin/auth"
)

func main() {
	cli.Execute()
}

