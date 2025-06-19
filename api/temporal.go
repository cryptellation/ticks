package api

import (
	"github.com/cryptellation/runtime"
	"github.com/cryptellation/ticks/pkg/tick"
	"github.com/google/uuid"
)

const (
	// WorkerTaskQueueName is the name of the task queue for the cryptellation worker.
	WorkerTaskQueueName = "CryptellationticksTaskQueue"
)

const (
	// RegisterForTicksListeningWorkflowName is the name of the workflow to register
	// for ticks reception through a callback workflow.
	RegisterForTicksListeningWorkflowName = "RegisterForTicksListeningWorkflow"
)

type (
	// RegisterForTicksListeningWorkflowParams is the parameters of the
	// RegisterForTicksListening workflow.
	RegisterForTicksListeningWorkflowParams struct {
		RequesterID uuid.UUID
		Exchange    string
		Pair        string
		Callback    runtime.CallbackWorkflow
	}

	// ListenToTicksCallbackWorkflowParams is the parameters of the
	// RegisterForTicksListening callback workflow.
	ListenToTicksCallbackWorkflowParams struct {
		RequesterID uuid.UUID
		Tick        tick.Tick
	}

	// RegisterForTicksListeningWorkflowResults is the results of the
	// RegisterForTicksListening workflow.
	RegisterForTicksListeningWorkflowResults struct {
	}
)

const (
	// UnregisterFromTicksListeningWorkflowName is the name of the workflow to register
	// for ticks reception through a callback workflow.
	UnregisterFromTicksListeningWorkflowName = "UnregisterFromTicksListeningWorkflow"
)

type (
	// UnregisterFromTicksListeningWorkflowParams is the parameters of the
	// UnregisterFromTicksListening workflow.
	UnregisterFromTicksListeningWorkflowParams struct {
		RequesterID uuid.UUID
		Exchange    string
		Pair        string
	}

	// UnregisterFromTicksListeningWorkflowResults is the results of the
	// UnregisterFromTicksListening workflow.
	UnregisterFromTicksListeningWorkflowResults struct{}
)

const (
	// ServiceInfoWorkflowName is the name of the workflow to get the service info.
	ServiceInfoWorkflowName = "ServiceInfoWorkflow"
)

type (
	// ServiceInfoParams contains the parameters of the service info workflow.
	ServiceInfoParams struct{}

	// ServiceInfoResults contains the result of the service info workflow.
	ServiceInfoResults struct {
		Version string
	}
)
