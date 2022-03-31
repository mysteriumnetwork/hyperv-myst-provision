package vbox

import "errors"

// Hyper-V networking constants
const (

	StateDisabled = 3
	StateEnabled  = 2
)

var (
	errEmptyIP = errors.New("could not resolve IP address")
)
