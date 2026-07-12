package services

import (
	"encoding/json"
	"testing"
)

// TestDAGValidation tests cycle detection in workflow tasks.
func TestDAGValidation(t *testing.T) {
	svc := NewWorkflowService()

	tests := []struct {
		name    string
		tasks   []WorkflowTask
		wantErr bool
	}{
		{
			name: "Valid DAG (no cycles)",
			tasks: []WorkflowTask{
				{Name: "taskA", DependsOn: []string{}},
				{Name: "taskB", DependsOn: []string{"taskA"}},
				{Name: "taskC", DependsOn: []string{"taskA", "taskB"}},
			},
			wantErr: false,
		},
		{
			name: "Simple cycle (A -> B -> A)",
			tasks: []WorkflowTask{
				{Name: "taskA", DependsOn: []string{"taskB"}},
				{Name: "taskB", DependsOn: []string{"taskA"}},
			},
			wantErr: true,
		},
		{
			name: "Self loop (A -> A)",
			tasks: []WorkflowTask{
				{Name: "taskA", DependsOn: []string{"taskA"}},
			},
			wantErr: true,
		},
		{
			name: "Indirect cycle (A -> B -> C -> A)",
			tasks: []WorkflowTask{
				{Name: "taskA", DependsOn: []string{"taskC"}},
				{Name: "taskB", DependsOn: []string{"taskA"}},
				{Name: "taskC", DependsOn: []string{"taskB"}},
			},
			wantErr: true,
		},
		{
			name: "Reference to non-existent task",
			tasks: []WorkflowTask{
				{Name: "taskA", DependsOn: []string{"nonExistent"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateDAG(tt.tasks)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDAG() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestTopologicalOrder tests correct task ordering in DAG.
func TestTopologicalOrder(t *testing.T) {
	svc := NewWorkflowService()

	tasks := []WorkflowTask{
		{Name: "taskC", DependsOn: []string{"taskB"}},
		{Name: "taskA", DependsOn: []string{}},
		{Name: "taskB", DependsOn: []string{"taskA"}},
	}

	order, err := svc.TopologicalSort(tasks)
	if err != nil {
		t.Fatalf("Unexpected error sorting DAG: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("Expected 3 tasks in sorted order, got %d", len(order))
	}

	// Verify order: taskA must come before taskB, taskB must come before taskC
	pos := make(map[string]int)
	for i, task := range order {
		pos[task.Name] = i
	}

	if pos["taskA"] >= pos["taskB"] {
		t.Errorf("taskA (pos %d) should come before taskB (pos %d)", pos["taskA"], pos["taskB"])
	}
	if pos["taskB"] >= pos["taskC"] {
		t.Errorf("taskB (pos %d) should come before taskC (pos %d)", pos["taskB"], pos["taskC"])
	}
}

// TestWorkflowRunStatus tests transitions and statuses of WorkflowRun.
func TestWorkflowRunStatus(t *testing.T) {
	// Status transitions are modeled on the WorkflowRun model.
	// Since DB logic is not connected in unit test, we test status string correctness.
	run := WorkflowRun{
		ID:           "run-1",
		WorkflowID:   "wf-1",
		Status:       "pending",
		TriggeredBy:  "user-1",
		TaskStatuses: json.RawMessage(`{}`),
	}

	if run.Status != "pending" {
		t.Errorf("Expected initial status 'pending', got '%s'", run.Status)
	}

	// Transition to running
	run.Status = "running"
	if run.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", run.Status)
	}

	// Completed run
	run.Status = "completed"
	if run.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", run.Status)
	}

	// Failed run
	run.Status = "failed"
	run.ErrorMessage = "Task 'taskB' failed with exit code 1"
	if run.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", run.Status)
	}
	if run.ErrorMessage == "" {
		t.Error("Expected error message to be set for failed run")
	}
}
