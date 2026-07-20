package cli

import (
	"errors"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/escalation"
)

// Error is a CLI error carrying the process exit code and a stable machine code
// (the JSON error envelope's "code" field). The exit codes are an API asserted
// on by callers — see the exit-code table in the interface spec.
type Error struct {
	Exit    int
	Code    string
	Message string
	ID      string
}

func (e *Error) Error() string { return e.Message }

func usageErr(msg string) *Error {
	return &Error{Exit: 2, Code: "usage", Message: msg}
}

func internalErr(msg string) *Error {
	return &Error{Exit: 1, Code: "error", Message: msg}
}

func validationErr(msg string) *Error {
	return &Error{Exit: 3, Code: "validation", Message: msg}
}

func notFoundErr(msg, id string) *Error {
	return &Error{Exit: 4, Code: "not_found", Message: msg, ID: id}
}

func conflictErr(msg string) *Error {
	return &Error{Exit: 5, Code: "conflict", Message: msg}
}

func preconditionErr(msg string) *Error {
	return &Error{Exit: 6, Code: "precondition", Message: msg}
}

// wrapStoreErr translates backlog/escalation package errors into a *Error with
// the right exit code.
func wrapStoreErr(err error) *Error {
	if err == nil {
		return nil
	}
	var ve *backlog.ValidationError
	if errors.As(err, &ve) {
		return validationErr(ve.Msg)
	}
	var eve *escalation.ValidationError
	if errors.As(err, &eve) {
		return validationErr(eve.Msg)
	}
	var pe *escalation.PreconditionError
	if errors.As(err, &pe) {
		return preconditionErr(pe.Msg)
	}
	if errors.Is(err, backlog.ErrNotFound) || errors.Is(err, escalation.ErrNotFound) {
		return notFoundErr(err.Error(), "")
	}
	if errors.Is(err, backlog.ErrConflict) {
		return conflictErr(err.Error())
	}
	return internalErr(err.Error())
}
