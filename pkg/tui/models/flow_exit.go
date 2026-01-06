// Package models provides Bubble Tea models for the TUI interface.
package models

// FlowExitBehavior defines how a flow signals exit to its caller.
type FlowExitBehavior int

const (
	// FlowExitQuit exits the Bubble Tea program (default).
	FlowExitQuit FlowExitBehavior = iota
	// FlowExitMessage emits an exit message for embedding.
	FlowExitMessage
)

// FlowExitReason describes why a flow exited.
type FlowExitReason int

const (
	// FlowExitCompleted indicates the flow finished successfully.
	FlowExitCompleted FlowExitReason = iota
	// FlowExitCancelled indicates the flow was cancelled by the user.
	FlowExitCancelled
	// FlowExitDeclined indicates the user declined the confirmation.
	FlowExitDeclined
	// FlowExitError indicates the flow exited after an error.
	FlowExitError
)

// DownFlowExitMsg is emitted when a down flow exits in embedded mode.
type DownFlowExitMsg struct {
	Reason FlowExitReason
	Err    error
}

// UpFlowExitMsg is emitted when an up flow exits in embedded mode.
type UpFlowExitMsg struct {
	Reason FlowExitReason
	Err    error
}
