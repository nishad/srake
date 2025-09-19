package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nishad/srake/internal/config"
	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage srake configuration",
	Long:  `Manage srake configuration including paths, settings, and preferences.`,
}

var configPathsCmd = &cobra.Command{
	Use:   "paths",
	Short: "Show all active paths",
	Long: `Display all paths used by srake including configuration, data, cache,
and state directories. Also shows any environment variable overrides.`,
	RunE: runConfigPaths,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings.`,
	RunE:  runConfigShow,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration",
	Long: `Create a default configuration file in the appropriate location.

This will create a config file at ~/.config/srake/config.yaml with
sensible defaults. If a config file already exists, use --force to
overwrite it.`,
	Example: `  # Create default config
  srake config init

  # Force overwrite existing config
  srake config init --force`,
	RunE: runConfigInit,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration file",
	Long: `Open the configuration file in your default editor.

Uses the EDITOR environment variable, falling back to vi if not set.`,
	RunE: runConfigEdit,
}

var (
	configForce bool
)

func init() {
	configInitCmd.Flags().BoolVar(&configForce, "force", false, "Overwrite existing configuration")

	configCmd.AddCommand(configPathsCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configEditCmd)
}

func runConfigPaths(cmd *cobra.Command, args []string) error {
	p := paths.GetPaths()

	printInfo("SRake Paths")
	fmt.Println(colorize(colorGray, "────────────────────────────────────────"))

	// Show base paths
	fmt.Printf("%s\n", colorize(colorBold, "Base Directories:"))
	fmt.Printf("  Config:   %s\n", colorize(colorCyan, p.ConfigDir))
	fmt.Printf("  Data:     %s\n", colorize(colorCyan, p.DataDir))
	fmt.Printf("  Cache:    %s\n", colorize(colorCyan, p.CacheDir))
	fmt.Printf("  State:    %s\n", colorize(colorCyan, p.StateDir))

	fmt.Println()
	fmt.Printf("%s\n", colorize(colorBold, "Specific Paths:"))
	fmt.Printf("  Database:  %s\n", colorize(colorCyan, paths.GetDatabasePath()))
	fmt.Printf("  Index:     %s\n", colorize(colorCyan, paths.GetIndexPath()))
	fmt.Printf("  Models:    %s\n", colorize(colorCyan, paths.GetModelsPath()))
	fmt.Printf("  Downloads: %s\n", colorize(colorCyan, paths.GetDownloadsPath()))
	fmt.Printf("  Resume:    %s\n", colorize(colorCyan, paths.GetResumePath()))

	// Show environment variables if set
	envVars := []struct {
		name string
		desc string
	}{
		{"SRAKE_CONFIG_HOME", "Override config directory"},
		{"SRAKE_DATA_HOME", "Override data directory"},
		{"SRAKE_CACHE_HOME", "Override cache directory"},
		{"SRAKE_STATE_HOME", "Override state directory"},
		{"SRAKE_DB_PATH", "Override database path"},
		{"SRAKE_INDEX_PATH", "Override index path"},
		{"SRAKE_MODELS_PATH", "Override models path"},
	}

	hasEnv := false
	for _, env := range envVars {
		if os.Getenv(env.name) != "" {
			hasEnv = true
			break
		}
	}

	if hasEnv {
		fmt.Println()
		fmt.Printf("%s\n", colorize(colorBold, "Environment Variables:"))
		for _, env := range envVars {
			if val := os.Getenv(env.name); val != "" {
				fmt.Printf("  %s = %s\n",
					colorize(colorYellow, env.name),
					colorize(colorCyan, val))
				if verbose {
					fmt.Printf("    %s\n", colorize(colorGray, env.desc))
				}
			}
		}
	}

	// Check if paths exist
	fmt.Println()
	fmt.Printf("%s\n", colorize(colorBold, "Path Status:"))

	pathChecks := []struct {
		name string
		path string
	}{
		{"Config Dir", p.ConfigDir},
		{"Data Dir", p.DataDir},
		{"Database", paths.GetDatabasePath()},
		{"Index", paths.GetIndexPath()},
	}

	for _, check := range pathChecks {
		if _, err := os.Stat(check.path); err == nil {
			fmt.Printf("  %-12s %s\n", check.name+":", colorize(colorGreen, "✓ exists"))
		} else {
			fmt.Printf("  %-12s %s\n", check.name+":", colorize(colorGray, "✗ not found"))
		}
	}

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	configPath := config.GetConfigPath()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	printInfo("Configuration")
	fmt.Println(colorize(colorGray, "────────────────────────────────────────"))

	fmt.Printf("%s %s\n", colorize(colorBold, "Config File:"), configPath)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println(colorize(colorYellow, "  (using defaults - no config file found)"))
	}

	fmt.Println()

	// Marshal config to YAML for display
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	// Pretty print with syntax highlighting
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			fmt.Println()
			continue
		}

		// Simple syntax highlighting
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			// Top-level keys
			fmt.Println(colorize(colorBold, line))
		} else if strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			indent := len(line) - len(strings.TrimLeft(line, " "))
			fmt.Printf("%s%s: %s\n",
				strings.Repeat(" ", indent),
				colorize(colorCyan, strings.TrimSpace(parts[0])),
				colorize(colorGreen, parts[1]))
		} else {
			fmt.Println(line)
		}
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	configPath := filepath.Join(paths.GetPaths().ConfigDir, "config.yaml")

	// Check if config exists
	if _, err := os.Stat(configPath); err == nil && !configForce {
		printWarning("Configuration already exists at %s", configPath)
		fmt.Println("Use --force to overwrite")
		return nil
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Save to file
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printSuccess("Configuration created at %s", configPath)

	// Show the config
	fmt.Println()
	return runConfigShow(cmd, args)
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	configPath := config.GetConfigPath()

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		printWarning("No configuration file found")
		fmt.Printf("Create one with: srake config init\n")
		return nil
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default to vi
	}

	printInfo("Opening %s in %s", configPath, editor)

	// Execute editor
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	// Validate the edited config
	if _, err := config.Load(configPath); err != nil {
		printError("Configuration validation failed: %v", err)
		return err
	}

	printSuccess("Configuration updated successfully")
	return nil
}
