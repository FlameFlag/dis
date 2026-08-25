package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/4evy/dis/internal/config"
	"github.com/4evy/dis/internal/convert"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	nushellCommentColumn = 30
	pflagTypeString      = "string"
	nushellTypeString    = "string"
	nushellTypeInt       = "int"
	nushellTypeFloat     = "float"
)

type nushellCompleter struct {
	flag   string
	name   string
	values func() []string
}

var nushellCompleters = []nushellCompleter{
	{flagVideoCodec, "nu-complete dis video-codec", convert.CodecNames},
	{flagResolution, "nu-complete dis resolution", convert.ResolutionStrings},
	{flagPreset, "nu-complete dis preset", configPresetNames},
}

func configPresetNames() []string { return config.PresetNames(nil) }

func addCompletionCommands(root *cobra.Command) {
	completion := &cobra.Command{
		Use:   "completion [bash|zsh|fish|nushell]",
		Short: "Generate shell completion scripts",
		Long:  "Generate completion scripts for bash, zsh, fish, or nushell.",
	}
	shells := []struct {
		name     string
		generate func(io.Writer) error
	}{
		{"bash", func(writer io.Writer) error {
			return root.GenBashCompletionV2(writer, true)
		}},
		{"zsh", root.GenZshCompletion},
		{"fish", func(writer io.Writer) error {
			return root.GenFishCompletion(writer, true)
		}},
		{"nushell", func(writer io.Writer) error {
			return generateNushellCompletion(root, writer)
		}},
	}
	for _, shell := range shells {
		completion.AddCommand(&cobra.Command{
			Use:   shell.name,
			Short: "Generate " + shell.name + " completions",
			Args:  cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error {
				return shell.generate(command.OutOrStdout())
			},
		})
	}
	root.AddCommand(completion)
}

func generateNushellCompletion(command *cobra.Command, writer io.Writer) error {
	for _, completer := range nushellCompleters {
		if err := writeNushellCompleter(
			writer,
			completer.name,
			completer.values(),
		); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(
		writer,
		"export extern \"%s\" [\n  ...input: string              # Input URLs or file paths\n",
		command.Name(),
	); err != nil {
		return err
	}
	var writeErr error
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if writeErr == nil {
			writeErr = writeNushellFlag(writer, flag)
		}
	})
	if writeErr != nil {
		return writeErr
	}
	_, err := fmt.Fprint(writer, "]\n")
	return err
}

func writeNushellCompleter(writer io.Writer, name string, values []string) error {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = fmt.Sprintf("%q", value)
	}
	_, err := fmt.Fprintf(
		writer,
		"def \"%s\" [] {\n  [%s]\n}\n\n",
		name,
		strings.Join(quoted, " "),
	)
	return err
}

func writeNushellFlag(writer io.Writer, flag *pflag.Flag) error {
	if flag.Hidden {
		return nil
	}
	var builder strings.Builder
	builder.WriteString("  --")
	builder.WriteString(flag.Name)
	if flag.Shorthand != "" {
		fmt.Fprintf(&builder, " (-%s)", flag.Shorthand)
	}
	if valueType := nushellFlagType(flag); valueType != "" {
		builder.WriteString(": ")
		builder.WriteString(valueType)
		if index := slices.IndexFunc(
			nushellCompleters,
			func(completer nushellCompleter) bool {
				return completer.flag == flag.Name
			},
		); index >= 0 {
			fmt.Fprintf(&builder, "@\"%s\"", nushellCompleters[index].name)
		}
	}
	line := builder.String()
	if flag.Usage != "" {
		padding := max(nushellCommentColumn-len(line), 1)
		line += strings.Repeat(" ", padding) + "# " + flag.Usage
	}
	_, err := fmt.Fprintln(writer, line)
	return err
}

func nushellFlagType(flag *pflag.Flag) string {
	switch flag.Value.Type() {
	case pflagTypeString:
		return nushellTypeString
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return nushellTypeInt
	case "float32", "float64":
		return nushellTypeFloat
	case "bool":
		return ""
	default:
		return nushellTypeString
	}
}
