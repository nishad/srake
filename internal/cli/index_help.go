package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SetupIndexHelp configures the search index command to display its flags properly
func SetupIndexHelp(cmd *cobra.Command) {
	originalHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// First print the original help without flags
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Hidden = true
		})
		originalHelpFunc(cmd, args)
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			flag.Hidden = false
		})

		// Now print grouped flags for index command
		fmt.Println("\nFlags:")

		printFlagGroup(cmd, "INDEX OPERATIONS", []string{
			"build",
			"rebuild",
			"verify",
			"stats",
			"resume",
		})

		printFlagGroup(cmd, "INDEX OPTIONS", []string{
			"batch-size",
			"workers",
			"path",
			"with-embeddings",
			"embedding-model",
			"progress",
			"progress-file",
			"checkpoint-dir",
		})

		printFlagGroup(cmd, "GLOBAL OPTIONS", []string{
			"help", "h",
			"verbose", "v",
			"quiet", "q",
			"no-color",
		})

		// Print environment variables section
		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  SRAKE_DB_PATH          Path to the SRA metadata database")
		fmt.Println("  SRAKE_INDEX_PATH       Path to the search index directory")
		fmt.Println("  SRAKE_CONFIG_DIR       Configuration directory (default: ~/.config/srake)")
		fmt.Println("  SRAKE_DATA_DIR         Data directory (default: ~/.local/share/srake)")
		fmt.Println("  SRAKE_CACHE_DIR        Cache directory (default: ~/.cache/srake)")
		fmt.Println("  SRAKE_MODEL_VARIANT    Model variant for embeddings (full|quantized)")
		fmt.Println("  NO_COLOR               Disable colored output")
		fmt.Println("\nFor more information, visit: https://github.com/nishad/srake")
	})
}

// printIndexFlagGroup prints a group of flags with a header (reused from help.go)
func printIndexFlagGroup(cmd *cobra.Command, groupName string, flagNames []string) {
	var flags []*pflag.Flag
	flagMap := make(map[string]bool)

	for _, name := range flagNames {
		flagMap[name] = true
		if flag := cmd.Flags().Lookup(name); flag != nil && !flag.Hidden {
			flags = append(flags, flag)
		}
	}

	// Skip empty groups
	if len(flags) == 0 {
		return
	}

	fmt.Printf("\n%s:\n", groupName)

	// Find maximum flag name length for alignment
	maxLen := 0
	for _, flag := range flags {
		nameLen := len(flag.Name)
		if flag.Shorthand != "" {
			nameLen += 4 // ", -X"
		}
		if nameLen > maxLen {
			maxLen = nameLen
		}
	}
	maxLen += 4 // Add some padding

	// Print flags
	for _, flag := range flags {
		name := "--" + flag.Name
		if flag.Shorthand != "" {
			name = fmt.Sprintf("-%s, --%s", flag.Shorthand, flag.Name)
		}

		// Format the flag line
		padding := strings.Repeat(" ", maxLen-len(name))
		usage := flag.Usage

		// Add type and default value
		typeStr := ""
		switch flag.Value.Type() {
		case "string":
			if flag.DefValue != "" && flag.DefValue != "0" && flag.DefValue != "false" {
				typeStr = fmt.Sprintf(" string (default \"%s\")", flag.DefValue)
			} else {
				typeStr = " string"
			}
		case "int":
			if flag.DefValue != "0" {
				typeStr = fmt.Sprintf(" (default %s)", flag.DefValue)
			}
		case "bool":
			// Bool flags don't need type annotation
		default:
			typeStr = fmt.Sprintf(" %s", flag.Value.Type())
			if flag.DefValue != "" && flag.DefValue != "0" && flag.DefValue != "false" {
				typeStr = fmt.Sprintf(" %s (default %s)", flag.Value.Type(), flag.DefValue)
			}
		}

		fmt.Printf("  %s%s%s%s\n", name, padding, usage, typeStr)
	}
}