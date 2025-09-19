package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SetupGroupedHelp configures a command to display flags grouped by category
func SetupGroupedHelp(cmd *cobra.Command) {
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

		// Now print grouped flags
		fmt.Println("\nFlags:")
		printFlagGroup(cmd, "FILTER OPTIONS", []string{
			"organism", "o",
			"platform",
			"library-strategy",
			"library-source",
			"library-selection",
			"library-layout",
			"study-type",
			"instrument-model",
			"date-from",
			"date-to",
			"spots-min",
			"spots-max",
			"bases-min",
			"bases-max",
		})

		printFlagGroup(cmd, "QUALITY CONTROL", []string{
			"similarity-threshold", "s",
			"min-score", "m",
			"top-percentile", "t",
			"show-confidence", "c",
			"hybrid-weight", "w",
		})

		printFlagGroup(cmd, "OUTPUT OPTIONS", []string{
			"format", "f",
			"output",
			"limit", "l",
			"offset",
			"no-header",
			"fields",
		})

		printFlagGroup(cmd, "SEARCH MODES", []string{
			"fuzzy",
			"exact",
			"stats",
			"facets",
			"highlight",
			"advanced",
			"search-mode",
			"no-fts",
			"no-vectors",
		})

		printFlagGroup(cmd, "GLOBAL OPTIONS", []string{
			"help", "h",
			"verbose", "v",
			"quiet", "q",
			"no-color",
			"debug",
		})

		// Print environment variables section
		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  SRAKE_DB_PATH          Path to the SRA metadata database")
		fmt.Println("  SRAKE_INDEX_PATH       Path to the search index directory")
		fmt.Println("  SRAKE_CONFIG_DIR       Configuration directory (default: ~/.config/srake)")
		fmt.Println("  SRAKE_DATA_DIR         Data directory (default: ~/.local/share/srake)")
		fmt.Println("  SRAKE_CACHE_DIR        Cache directory (default: ~/.cache/srake)")
		fmt.Println("  NO_COLOR               Disable colored output")
		fmt.Println("\nFor more information, visit: https://github.com/nishad/srake")
	})
}

// printFlagGroup prints a group of flags with a header
func printFlagGroup(cmd *cobra.Command, groupName string, flagNames []string) {
	var flags []*pflag.Flag
	flagMap := make(map[string]bool)

	for _, name := range flagNames {
		flagMap[name] = true
		if flag := cmd.Flags().Lookup(name); flag != nil && !flag.Hidden {
			flags = append(flags, flag)
		}
	}

	if len(flags) == 0 {
		return
	}

	fmt.Printf("\n%s:\n", groupName)
	for _, flag := range flags {
		shorthand := ""
		if flag.Shorthand != "" {
			shorthand = fmt.Sprintf("-%s, ", flag.Shorthand)
		}

		// Format the flag line
		flagLine := fmt.Sprintf("  %s--%s", shorthand, flag.Name)

		// Add type information
		typeStr := ""
		switch flag.Value.Type() {
		case "string":
			if flag.DefValue != "" && flag.DefValue != "[]" {
				typeStr = fmt.Sprintf(" string (default %q)", flag.DefValue)
			} else {
				typeStr = " string"
			}
		case "int", "int32", "int64":
			if flag.DefValue != "0" {
				typeStr = fmt.Sprintf(" int (default %s)", flag.DefValue)
			} else {
				typeStr = " int"
			}
		case "float32", "float64":
			typeStr = fmt.Sprintf(" float (default %s)", flag.DefValue)
		case "bool":
			typeStr = ""
		default:
			if flag.DefValue != "" && flag.DefValue != "[]" {
				typeStr = fmt.Sprintf(" (default %s)", flag.DefValue)
			}
		}

		// Ensure proper alignment
		padding := 45 - len(flagLine) - len(typeStr)
		if padding < 1 {
			padding = 1
		}

		fmt.Printf("%s%s%s%s\n", flagLine, typeStr, strings.Repeat(" ", padding), flag.Usage)
	}
}