package errors

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrTaskNotFound      = errors.New("task not found")
	ErrInstanceNotFound  = errors.New("instance not found")
	ErrFlowNotFound      = errors.New("flow not found")
	ErrExecutionFailed   = errors.New("execution failed")
	ErrParseParamsFailed = errors.New("parse params failed")
)
