package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"dis/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|nushell]",
	Short: "Generate shell completion scripts",
	Long:  `Generate completion scripts for bash, zsh, fish, or nushell.`,
}

var bashCompletionCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletionV2(os.Stdout, true)
	},
}

var zshCompletionCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var fishCompletionCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(os.Stdout, true)
	},
}

var nushellCompletionCmd = &cobra.Command{
	Use:   "nushell",
	Short: "Generate nushell completions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return genNushellCompletion(rootCmd, os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(bashCompletionCmd, zshCompletionCmd, fishCompletionCmd, nushellCompletionCmd)
	rootCmd.AddCommand(completionCmd)
}

// nushellCompleter maps flag names to their nushell completer function names.
var nushellCompleters = map[string]string{
	"video-codec": "nu-complete dis video-codec",
	"resolution":  "nu-complete dis resolution",
	"preset":      "nu-complete dis preset",
}

func genNushellCompletion(cmd *cobra.Command, w io.Writer) error {
	// Emit completer functions.
	codecNames := config.CodecNames()
	writeNushellCompleter(w, "nu-complete dis video-codec", codecNames)
	writeNushellCompleter(w, "nu-complete dis resolution",
		[]string{"144p", "240p", "360p", "480p", "720p", "1080p", "1440p", "2160p"})
	writeNushellCompleter(w, "nu-complete dis preset",
		[]string{"discord", "discord-nitro", "twitter", "telegram"})

	// Emit the extern declaration.
	fmt.Fprintf(w, "export extern \"%s\" [\n", cmd.Name())
	fmt.Fprintf(w, "  ...input: string              # Input URLs or file paths\n")

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		writeNushellFlag(w, f, cmd.Name())
	})

	fmt.Fprintf(w, "]\n")
	return nil
}

func writeNushellCompleter(w io.Writer, name string, values []string) {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	fmt.Fprintf(w, "def \"%s\" [] {\n", name)
	fmt.Fprintf(w, "  [%s]\n", strings.Join(quoted, " "))
	fmt.Fprintf(w, "}\n\n")
}

func writeNushellFlag(w io.Writer, f *pflag.Flag, cmdName string) {
	if f.Hidden {
		return
	}

	var sb strings.Builder
	sb.WriteString("  --")
	sb.WriteString(f.Name)

	if f.Shorthand != "" {
		sb.WriteString(fmt.Sprintf(" (-%s)", f.Shorthand))
	}

	nuType := nushellFlagType(f)
	if nuType != "" {
		sb.WriteString(": ")
		sb.WriteString(nuType)
		if completer, ok := nushellCompleters[f.Name]; ok {
			sb.WriteString(fmt.Sprintf("@\"%s\"", completer))
		}
	}

	// Pad and add usage as comment.
	line := sb.String()
	if f.Usage != "" {
		pad := 30 - len(line)
		if pad < 1 {
			pad = 1
		}
		line += strings.Repeat(" ", pad) + "# " + f.Usage
	}

	fmt.Fprintf(w, "%s\n", line)
}

// nushellFlagType returns the nushell type annotation for a pflag flag.
// Returns empty string for bool flags (they are switches in nushell).
func nushellFlagType(f *pflag.Flag) string {
	switch f.Value.Type() {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "int"
	case "float32", "float64":
		return "float"
	case "bool":
		return "" // nushell switch, no type annotation
	default:
		return "string"
	}
}
