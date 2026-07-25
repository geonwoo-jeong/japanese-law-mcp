package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}使用方法:
  {{.UseLine}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [コマンド]{{end}}{{if .HasAvailableSubCommands}}

利用可能なコマンド:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
フラグ:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}`

func configureHelp(root *cobra.Command) {
	root.SetHelpTemplate(helpTemplate)
	root.SetUsageTemplate(helpTemplate)
	root.InitDefaultHelpFlag()
	if helpFlag := root.Flags().Lookup("help"); helpFlag != nil {
		helpFlag.Usage = "ヘルプを表示します"
	}

	helpCommand := &cobra.Command{
		Use:   "help [command]",
		Short: "コマンドのヘルプを表示します",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return usageError("help に指定できるコマンドは一つだけです。")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			target := root
			if len(args) == 1 {
				found, _, err := root.Find(args)
				if err != nil || found == root {
					return usageError(fmt.Sprintf("コマンド %q は見つかりません。", args[0]))
				}
				target = found
			}
			return target.Help()
		},
	}
	root.SetHelpCommand(helpCommand)
}
