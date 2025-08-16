package cadence

import (
	"fmt"
	"time"
)

// CadenceClient represents a Cadence workflow client
type CadenceClient struct {
	client      Client
	domain      string
	config      *CadenceConfig
	metrics     *CadenceMetrics
}

// Client represents a Cadence client interface
type Client interface {
	StartWorkflow(ctx interface{}, options interface{}, workflowType string, args ...interface{}) (*WorkflowExecution, error)
	GetWorkflow(ctx interface{}, workflowID string, runID string) (*WorkflowExecution, error)
	SignalWorkflow(ctx interface{}, workflowID string, runID string, signalName string, args ...interface{}) error
	CancelWorkflow(ctx interface{}, workflowID string, runID string) error
	TerminateWorkflow(ctx interface{}, workflowID string, runID string, reason string) error
}

// WorkflowExecution represents a workflow execution
type WorkflowExecution struct {
	ID        string
	RunID     string
	WorkflowID string
	Status    string
	StartTime time.Time
	EndTime   time.Time
}

// CadenceConfig represents configuration for Cadence client
type CadenceConfig struct {
	Domain              string
	TaskList            string
	WorkflowID          string
	ExecutionTimeout    time.Duration
	DecisionTimeout     time.Duration
	EnableRetry         bool
	MaxRetries          int
	RetryDelay          time.Duration
}

// CadenceMetrics represents metrics for Cadence operations
type CadenceMetrics struct {
	WorkflowsStarted    int64
	WorkflowsCompleted  int64
	WorkflowsFailed     int64
	SignalsSent         int64
	AverageExecutionTime time.Duration
	LastOperation       time.Time
}

// NewCadenceClient creates a new Cadence client
func NewCadenceClient(client Client, config *CadenceConfig) *CadenceClient {
	if config == nil {
		config = &CadenceConfig{
			Domain:           "cloudbridge",
			TaskList:         "client-tasks",
			ExecutionTimeout: 1 * time.Hour,
			DecisionTimeout:  1 * time.Minute,
			EnableRetry:      true,
			MaxRetries:       3,
			RetryDelay:       5 * time.Second,
		}
	}

	return &CadenceClient{
		client:  client,
		domain:  config.Domain,
		config:  config,
		metrics: &CadenceMetrics{},
	}
}

// StartWorkflow starts a new workflow execution
func (cc *CadenceClient) StartWorkflow(ctx interface{}, workflowType string, input interface{}) (*WorkflowExecution, error) {
	workflowOptions := &WorkflowOptions{
		ID:                              cc.config.WorkflowID,
		TaskList:                        cc.config.TaskList,
		ExecutionStartToCloseTimeout:    cc.config.ExecutionTimeout,
		DecisionTaskStartToCloseTimeout: cc.config.DecisionTimeout,
		RetryPolicy: &RetryPolicy{
			InitialInterval:    cc.config.RetryDelay,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Minute,
			MaximumAttempts:    cc.config.MaxRetries,
		},
	}

	execution, err := cc.client.StartWorkflow(ctx, workflowOptions, workflowType, input)
	if err != nil {
		cc.metrics.WorkflowsFailed++
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	cc.metrics.WorkflowsStarted++
	cc.metrics.LastOperation = time.Now()

	return execution, nil
}

// GetWorkflow retrieves a workflow execution
func (cc *CadenceClient) GetWorkflow(ctx interface{}, workflowID, runID string) (*WorkflowExecution, error) {
	execution, err := cc.client.GetWorkflow(ctx, workflowID, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return execution, nil
}

// SignalWorkflow sends a signal to a workflow
func (cc *CadenceClient) SignalWorkflow(ctx interface{}, workflowID, runID, signalName string, args ...interface{}) error {
	err := cc.client.SignalWorkflow(ctx, workflowID, runID, signalName, args...)
	if err != nil {
		return fmt.Errorf("failed to signal workflow: %w", err)
	}

	cc.metrics.SignalsSent++
	cc.metrics.LastOperation = time.Now()

	return nil
}

// CancelWorkflow cancels a workflow execution
func (cc *CadenceClient) CancelWorkflow(ctx interface{}, workflowID, runID string) error {
	err := cc.client.CancelWorkflow(ctx, workflowID, runID)
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	cc.metrics.LastOperation = time.Now()
	return nil
}

// TerminateWorkflow terminates a workflow execution
func (cc *CadenceClient) TerminateWorkflow(ctx interface{}, workflowID, runID, reason string) error {
	err := cc.client.TerminateWorkflow(ctx, workflowID, runID, reason)
	if err != nil {
		return fmt.Errorf("failed to terminate workflow: %w", err)
	}

	cc.metrics.LastOperation = time.Now()
	return nil
}

// GetMetrics returns Cadence metrics
func (cc *CadenceClient) GetMetrics() *CadenceMetrics {
	return cc.metrics
}

// WorkflowOptions represents options for starting a workflow
type WorkflowOptions struct {
	ID                              string
	TaskList                        string
	ExecutionStartToCloseTimeout    time.Duration
	DecisionTaskStartToCloseTimeout time.Duration
	RetryPolicy                     *RetryPolicy
}

// RetryPolicy represents a retry policy
type RetryPolicy struct {
	InitialInterval    time.Duration
	BackoffCoefficient float64
	MaximumInterval    time.Duration
	MaximumAttempts    int
}
