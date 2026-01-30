package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"project-scaffold/internal/generator"
	"project-scaffold/internal/plugin"
)

var (
	flagStackKey   string
	flagDBKey      string
	flagNodeVariant string
	flagDocker     bool
	flagNoDocker   bool
	flagPlugins    string
)

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Initialize a new backend project scaffold",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exactly one argument: <project-name>")
		}
		name := strings.TrimSpace(args[0])
		if name == "" {
			color.New(color.FgRed).Fprintln(cmd.ErrOrStderr(), "Project name cannot be empty.")
			return errors.New("project-name cannot be empty")
		}
		if strings.ContainsAny(name, `<>:"/\|?*`) {
			color.New(color.FgRed).Fprintln(cmd.ErrOrStderr(), "Project name contains invalid path characters.")
			return errors.New("project-name contains invalid path characters")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := strings.TrimSpace(args[0])

		if flagDocker && flagNoDocker {
			color.New(color.FgRed).Fprintln(cmd.ErrOrStderr(), "You cannot use --docker and --no-docker at the same time.")
			return errors.New("cannot use --docker and --no-docker together")
		}

		stack := generator.Stack("")
		db := generator.Database("")
		nodeVariant := ""
		useDocker := false

		if strings.TrimSpace(flagStackKey) != "" {
			s, err := generator.ParseStackKey(flagStackKey)
			if err != nil {
				return err
			}
			stack = s
		}
		if strings.TrimSpace(flagDBKey) != "" {
			d, err := generator.ParseDatabaseKey(flagDBKey)
			if err != nil {
				return err
			}
			db = d
		}
		if strings.TrimSpace(flagNodeVariant) != "" {
			nodeVariant = strings.TrimSpace(strings.ToLower(flagNodeVariant))
			if nodeVariant != "js" && nodeVariant != "ts" {
				nodeVariant = "js"
			}
		}
		if flagDocker {
			useDocker = true
		} else if flagNoDocker {
			useDocker = false
		}

		stdinIsTTY := false
		if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			stdinIsTTY = true
		}
		nonInteractive := os.Getenv("SCAFFOLD_NON_INTERACTIVE") == "1" || os.Getenv("CI") == "true"

		qs := make([]*survey.Question, 0, 4)
		if stack == "" {
			qs = append(qs, &survey.Question{
				Name: "stack",
				Prompt: &survey.Select{
					Message: "Choose backend stack:",
					Options: []string{
						string(generator.StackGoGin),
						string(generator.StackNodeExpress),
					},
					Default: string(generator.StackGoGin),
				},
			})
		}
		if db == "" {
			qs = append(qs, &survey.Question{
				Name: "db",
				Prompt: &survey.Select{
					Message: "Choose database:",
					Options: []string{
						string(generator.DBPostgreSQL),
						string(generator.DBMongoDB),
						string(generator.DBSQLite),
					},
					Default: string(generator.DBPostgreSQL),
				},
			})
		}
		if stack == generator.StackNodeExpress && nodeVariant == "" {
			qs = append(qs, &survey.Question{
				Name: "nodeVariant",
				Prompt: &survey.Select{
					Message: "Node.js language:",
					Options: []string{"JavaScript", "TypeScript"},
					Default: "JavaScript",
				},
			})
		}
		if !flagDocker && !flagNoDocker {
			qs = append(qs, &survey.Question{
				Name: "docker",
				Prompt: &survey.Confirm{
					Message: "Use Docker?",
					Default: true,
				},
			})
		}

		didPrompt := len(qs) > 0
		if didPrompt {
			answers := struct {
				Stack  string `survey:"stack"`
				DB     string `survey:"db"`
				Docker bool   `survey:"docker"`
			}{}
			if err := survey.Ask(qs, &answers); err != nil {
				return err
			}
			if stack == "" {
				stack = generator.Stack(answers.Stack)
			}
			if db == "" {
				db = generator.Database(answers.DB)
			}
			if !flagDocker && !flagNoDocker {
				useDocker = answers.Docker
			}
		}
		if stack == generator.StackNodeExpress && nodeVariant == "" && stdinIsTTY && !nonInteractive {
			nodeVariantAnswers := struct {
				NodeVariant string `survey:"nodeVariant"`
			}{}
			if err := survey.Ask([]*survey.Question{{
				Name: "nodeVariant",
				Prompt: &survey.Select{
					Message: "Node.js language:",
					Options: []string{"JavaScript", "TypeScript"},
					Default: "JavaScript",
				},
			}}, &nodeVariantAnswers); err != nil {
				return err
			}
			if nodeVariantAnswers.NodeVariant == "TypeScript" {
				nodeVariant = "ts"
			} else {
				nodeVariant = "js"
			}
		}
		if stack == generator.StackNodeExpress && nodeVariant == "" {
			nodeVariant = "js"
		}

		pluginsSelected := parsePluginsFlag(flagPlugins)
		allFromFlags := stack != "" && db != "" && (flagDocker || flagNoDocker)
		if len(pluginsSelected) == 0 && didPrompt && !allFromFlags && stdinIsTTY && !nonInteractive {
			stackKey, err := generator.StackKey(stack)
			if err != nil {
				return err
			}
			effectiveStack := generator.EffectiveStackKey(stack, stackKey, nodeVariant)
			compatible := plugin.CompatibleWith(effectiveStack)
			if len(compatible) > 0 {
				qsPlugins := []*survey.Question{{
					Name: "plugins",
					Prompt: &survey.MultiSelect{
						Message: "Choose plugins (optional):",
						Options: compatible,
						Default: nil,
					},
				}}
				answersPlugins := struct {
					Plugins []string `survey:"plugins"`
				}{}
				if err := survey.Ask(qsPlugins, &answersPlugins); err != nil {
					return err
				}
				pluginsSelected = answersPlugins.Plugins
			}
		}

		targetDir := filepath.Join(".", projectName)
		if _, err := os.Stat(targetDir); err == nil {
			color.New(color.FgRed).Fprintln(cmd.ErrOrStderr(), "Folder already exists. Please choose a different project name.")
			return fmt.Errorf("target directory already exists: %s", targetDir)
		} else if !os.IsNotExist(err) {
			color.New(color.FgRed).Fprintf(cmd.ErrOrStderr(), "Could not access target directory: %v\n", err)
			return err
		}

		opts := generator.Options{
			ProjectName: projectName,
			Stack:       stack,
			Database:    db,
			UseDocker:   useDocker,
			Plugins:     pluginsSelected,
			NodeVariant: nodeVariant,
		}

		yellow := color.New(color.FgYellow)
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)

		yellow.Fprintf(cmd.OutOrStdout(), "[1/4] Creating project structure for %q\n", projectName)
		yellow.Fprintln(cmd.OutOrStdout(), "[2/4] Generating backend files")

		spin := spinner.New(spinner.CharSets[11], 120*time.Millisecond)
		spin.Writer = cmd.OutOrStdout()
		spin.Suffix = " Generating project files..."
		spin.Start()

		err := generator.Generate(targetDir, opts)
		spin.Stop()
		fmt.Fprintln(cmd.OutOrStdout())

		if err != nil {
			red.Fprintln(cmd.ErrOrStderr(), friendlyInitError(err))
			return err
		}

		yellow.Fprintln(cmd.OutOrStdout(), "[3/4] Writing environment configuration")
		yellow.Fprintln(cmd.OutOrStdout(), "[4/4] Finalizing project")

		green.Fprintf(cmd.OutOrStdout(), "✔ Project %q created successfully.\n", projectName)
		printNextSteps(cmd, opts)

		return nil
	},
}

func friendlyInitError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "template not found"):
		return "This stack + database combination is not available yet."
	default:
		return "Something went wrong while generating the project. Run with the same options again or check your environment."
	}
}

func printNextSteps(cmd *cobra.Command, opts generator.Options) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Next steps:")
	fmt.Fprintf(out, "  cd %s\n", opts.ProjectName)

	switch opts.Stack {
	case generator.StackGoGin:
		fmt.Fprintln(out, "  go mod tidy")
		fmt.Fprintln(out, "  go run ./cmd")
	case generator.StackNodeExpress:
		fmt.Fprintln(out, "  cp .env.example .env")
		fmt.Fprintln(out, "  npm install")
		fmt.Fprintln(out, "  npm run dev")
	}
}

func parsePluginsFlag(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func init() {
	initCmd.Flags().StringVar(&flagStackKey, "stack", "", "Stack key: go-gin | node-express (optional)")
	initCmd.Flags().StringVar(&flagDBKey, "db", "", "Database key: postgresql | mongodb | sqlite (optional)")
	initCmd.Flags().StringVar(&flagNodeVariant, "node-variant", "", "Node.js language: js | ts (optional, only for node-express)")
	initCmd.Flags().BoolVar(&flagDocker, "docker", false, "Generate Dockerfile and docker-compose.yml (skip prompt)")
	initCmd.Flags().BoolVar(&flagNoDocker, "no-docker", false, "Do not generate Docker files (skip prompt)")
	initCmd.Flags().StringVar(&flagPlugins, "plugins", "", "Comma-separated plugin names, e.g. auth (optional)")
}

