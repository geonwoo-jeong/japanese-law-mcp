package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/config"
	"github.com/spf13/cobra"
)

// Runner は、検証済みの設定でサーバーを実行する。
type Runner func(context.Context, config.Config) error

type commandOptions struct {
	in            io.Reader
	out           io.Writer
	errOut        io.Writer
	version       string
	userConfigDir func() (string, error)
	run           Runner
}

func newCommand(options commandOptions) *cobra.Command {
	var configFile string

	root := &cobra.Command{
		Use:                "japanese-law-mcp",
		Short:              "日本の公式法情報を MCP から利用できるようにします。",
		Long:               "Japanese Law MCP は、日本の公式法情報を検索・参照する MCP サーバーです。",
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableSuggestions: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return positionalArgumentsError(args)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(config.LoadOptions{
				Flags:         cmd.Flags(),
				ConfigFile:    configFile,
				UserConfigDir: options.userConfigDir,
			})
			if err != nil {
				return usageError(err.Error())
			}
			if options.run == nil {
				return runtimeError()
			}
			if err := options.run(cmd.Context(), cfg); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return runtimeError()
			}
			return nil
		},
	}

	root.SetIn(options.in)
	root.SetOut(options.out)
	root.SetErr(options.errOut)
	addConfigurationFlags(root, &configFile)
	configureHelp(root)
	root.AddCommand(newVersionCommand(options.version))

	return root
}

func addConfigurationFlags(root *cobra.Command, configFile *string) {
	flags := root.Flags()
	flags.StringVar(configFile, "config", "", "設定ファイルを明示します")
	flags.String("transport", "", "トランスポートを指定します")
	flags.Duration("request-timeout", 0, "外部リクエストのタイムアウトを指定します")
	flags.String("listen-address", "", "Streamable HTTP の待受先を指定します")
	flags.StringArray("allowed-origin", nil, "許可する HTTPS Origin を指定します（複数回指定できます）")
	flags.Bool("diagnostics", false, "一時診断を有効にします")
}

func newVersionCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "バージョン情報を表示します",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return positionalArgumentsError(args)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "japanese-law-mcp %s\n", version)
			if err != nil {
				return runtimeError()
			}
			return nil
		},
	}
}
