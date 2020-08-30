package mres

import (
	"errors"
)

var (
	//ErrInvalidArgument ...
	ErrInvalidArgument = errors.New("invalid argument")

	errReceivedCancellation = errors.New("received cancellation")
)
