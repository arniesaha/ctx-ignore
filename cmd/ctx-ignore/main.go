package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arniesaha/ctx-ignore/internal/generator"
	"github.com/arniesaha/ctx-ignore/internal/scanner"
	"github.com/arniesaha/ctx-ignore/internal/tokens"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ctx-ignore",
		Short: "Generate .claudeignore, .cursorignore, .contextignore for your repo",
		Long: `ctx-ignore scans your repository and generates ignore files to keep
AI coding tools focused on signal, not noise.`,
	}

	root.AddCommand(scanCmd())
	return root
}

func scanCmd() *cobra.Command {
	var dryRun bool
	var noClaude bool
	var noCursor bool
	var patchGitIgnore bool

	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a repo and generate ignore files",
		Long: `Scan a repository, classify files by LLM usefulness,
and generate .contextignore, .claudeignore, and .cursorignore files.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) > 0 {
				root = args[0]
			}
			abs, err := filepath.Abs(root)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			fmt.Printf("Scanning %s...\n\n", abs)

			s := scanner.New(abs)
			result, err := s.Scan()
			if err != nil {
				return fmt.Errorf("scanning: %w", err)
			}

			fmt.Printf("Scanning %d files...\n\n", result.TotalFiles)

			// Print noise report
			if len(result.NoisePatterns) == 0 {
				fmt.Println("✅ No significant noise detected.")
			} else {
				fmt.Println("NOISE (recommended to ignore):")
				for _, p := range result.NoisePatterns {
					icon := "🟡"
					if p.Level == scanner.NoiseLevelRed {
						icon = "🔴"
					}
					countStr := ""
					if p.Count > 1 {
						countStr = fmt.Sprintf(" (%d files)", p.Count)
					}
					fmt.Printf("  %s %-35s — %-12s  %s\n",
						icon,
						p.Pattern+countStr,
						tokens.FormatTokens(p.Tokens)+" tokens",
						string(p.Category),
					)
				}
				fmt.Println()
			}

			// Calculate signal tokens
			var signalTokens int64
			for _, s := range result.SignalItems {
				signalTokens += s.Tokens
			}

			// Print signal section (top 5)
			if len(result.SignalItems) > 0 {
				fmt.Println("SIGNAL (keeping in context):")
				max := 5
				if len(result.SignalItems) < max {
					max = len(result.SignalItems)
				}
				for _, si := range result.SignalItems[:max] {
					fmt.Printf("  ✅ %-35s — %s tokens\n",
						si.Path,
						tokens.FormatTokens(si.Tokens),
					)
				}
				if len(result.SignalItems) > 5 {
					fmt.Printf("  ... and %d more files\n", len(result.SignalItems)-5)
				}
				fmt.Println()
			}

			// Token reduction summary
			noiseTokens := result.TotalTokens - signalTokens
			reduction := 0.0
			if result.TotalTokens > 0 {
				reduction = float64(noiseTokens) / float64(result.TotalTokens) * 100
			}
			fmt.Printf("Before: %s tokens  →  After: %s tokens (%.1f%% reduction)\n\n",
				tokens.FormatTokens(result.TotalTokens),
				tokens.FormatTokens(signalTokens),
				reduction,
			)

			if dryRun {
				fmt.Println("--dry-run: no files written.")
				return nil
			}

			if len(result.NoisePatterns) == 0 {
				fmt.Println("Nothing to ignore — no files generated.")
				return nil
			}

			// Generate files
			opts := generator.DefaultOptions()
			opts.WriteClaudeIgnore = !noClaude
			opts.WriteCursorIgnore = !noCursor
			opts.PatchGitIgnore = patchGitIgnore
			opts.DryRun = dryRun
			opts.OutputDir = abs

			generated, err := generator.Generate(result, opts)
			if err != nil {
				return fmt.Errorf("generating files: %w", err)
			}

			fmt.Printf("Generated: %s\n", strings.Join(generated, ", "))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print report only, don't write files")
	cmd.Flags().BoolVar(&noClaude, "no-claude", false, "Skip .claudeignore generation")
	cmd.Flags().BoolVar(&noCursor, "no-cursor", false, "Skip .cursorignore generation")
	cmd.Flags().BoolVar(&patchGitIgnore, "patch-gitignore", false, "Also patch .gitignore")

	return cmd
}
