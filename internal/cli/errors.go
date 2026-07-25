package cli

import "fmt"

type commandError struct {
	code    int
	message string
}

func (e *commandError) Error() string {
	return e.message
}

func usageError(message string) error {
	return &commandError{
		code:    ExitUsage,
		message: message,
	}
}

func runtimeError() error {
	return &commandError{
		code:    ExitRuntime,
		message: "サーバーを実行できません。診断を有効にして実行環境を確認してください。",
	}
}

func invalidArgumentsError() error {
	return usageError("コマンドライン引数を解釈できません。--help で利用方法を確認してください。")
}

func positionalArgumentsError(args []string) error {
	return usageError(fmt.Sprintf("位置引数は使用できません: %q。--help で利用方法を確認してください。", args))
}
