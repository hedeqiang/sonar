// Package sonar provides a multi-chain event log monitoring SDK.
//
// Sonar — a deep probe for every on-chain event signal.
package sonar

import "errors"

var (
	// ErrChainNotFound is returned when operating on an unregistered chain.
	ErrChainNotFound = errors.New("sonar: chain not found")

	// ErrAlreadyRunning is returned when attempting to start a watcher that is already running.
	ErrAlreadyRunning = errors.New("sonar: watcher already running")

	// ErrNotRunning is returned when attempting to stop a watcher that is not running.
	ErrNotRunning = errors.New("sonar: watcher not running")

	// ErrShutdown is returned when operating on a shut-down Sonar instance.
	ErrShutdown = errors.New("sonar: instance has been shut down")

	// ErrInvalidAddress is returned when a malformed address string is encountered.
	ErrInvalidAddress = errors.New("sonar: invalid address")

	// ErrInvalidABI is returned when an ABI string cannot be parsed.
	ErrInvalidABI = errors.New("sonar: invalid ABI")

	// ErrDecode is returned when event log data cannot be decoded.
	ErrDecode = errors.New("sonar: decode failed")

	// ErrConnection is returned when the RPC/WebSocket connection fails.
	ErrConnection = errors.New("sonar: connection failed")

	// ErrChainAlreadyRegistered is returned when adding a chain that already exists.
	ErrChainAlreadyRegistered = errors.New("sonar: chain already registered")
)
