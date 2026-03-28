package cmd

type exitCodeError struct {
	code int
	msg  string
}

func (e *exitCodeError) Error() string {
	return e.msg
}

func (e *exitCodeError) ExitCode() int {
	return e.code
}

func newExitCodeError(code int, msg string) error {
	return &exitCodeError{
		code: code,
		msg:  msg,
	}
}
