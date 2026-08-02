package legalquerycandidateworker

import "errors"

const (
	FailureCodePreparedLoad   = 10
	FailureCodeRequestBinding = 11
	FailureCodeEvaluateBuild  = 12
	FailureCodeReportBinding  = 13
	FailureCodeAccept         = 14
	FailureCodeResultBind     = 15
	FailureCodeResultEncode   = 16
	FailureCodeResultDecode   = 17
	FailureCodeHandoffWrite   = 18
	FailureCodeTrackedReplay  = 19
	FailureCodeUnknown        = 22
)

type stageFailure interface {
	FailureExitCode() int
}

type stagedError struct {
	code int
	err  error
}

func (e stagedError) Error() string {
	return "候補評価 worker の内部段階が失敗しました"
}

func (e stagedError) Unwrap() error {
	return e.err
}

func (e stagedError) FailureExitCode() int {
	return e.code
}

func wrapFailure(code int, err error) error {
	if err == nil {
		return nil
	}
	var staged stageFailure
	if errors.As(err, &staged) {
		return err
	}
	return stagedError{code: code, err: err}
}

func FailureExitCode(err error) int {
	if err == nil {
		return 0
	}
	var staged stageFailure
	if errors.As(err, &staged) {
		code := staged.FailureExitCode()
		if isFailureExitCode(code) {
			return code
		}
	}
	return FailureCodeUnknown
}

func isFailureExitCode(code int) bool {
	return (code >= FailureCodePreparedLoad && code <= FailureCodeTrackedReplay) ||
		code == FailureCodeUnknown
}
