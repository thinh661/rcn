package services

import (
	"context"
	"errors"
	"testing"
)

// MockSparkJobRepository is a mock implementation of SparkJobRepository for testing.
type MockSparkJobRepository struct {
	GetJobStatusFunc func(ctx context.Context, jobID string) (string, error)
}

func (m *MockSparkJobRepository) GetJobStatus(ctx context.Context, jobID string) (string, error) {
	if m.GetJobStatusFunc != nil {
		return m.GetJobStatusFunc(ctx, jobID)
	}
	return "", errors.New("not implemented")
}

func TestParseSparkResource(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"1", 1, false},
		{"1g", 1024 * 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"1000m", 1000 * 1024 * 1024, false},
		{"1000M", 1000 * 1024 * 1024, false},
		{"500k", 500 * 1024, false},
		{"  2.5g ", 2684354560, false}, // 2.5 * 1024 * 1024 * 1024 = 2,684,354,560
		{"", 0, true},
		{"abc", 0, true},
		{"1x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := ParseSparkResource(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if res != tt.expected {
					t.Errorf("for input %q, expected %d, got %d", tt.input, tt.expected, res)
				}
			}
		})
	}
}

func TestMapSparkStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"RUNNING", "running"},
		{"running", "running"},
		{"FINISHED", "completed"},
		{"COMPLETED", "completed"},
		{"SUCCEEDED", "completed"},
		{"FAILED", "failed"},
		{"ERROR", "failed"},
		{"KILLED", "killed"},
		{"TERMINATED", "killed"},
		{"SUBMITTED", "pending"},
		{"PENDING", "pending"},
		{"ACCEPTED", "pending"},
		{"UNKNOWN_STATUS", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res := MapSparkStatus(tt.input)
			if res != tt.expected {
				t.Errorf("for input %q, expected %q, got %q", tt.input, tt.expected, res)
			}
		})
	}
}

func TestGetJobStatus(t *testing.T) {
	// Setup the mock repository
	mockRepo := &MockSparkJobRepository{
		GetJobStatusFunc: func(ctx context.Context, jobID string) (string, error) {
			if jobID == "job-123" {
				return "RUNNING", nil
			}
			if jobID == "job-456" {
				return "FINISHED", nil
			}
			return "", errors.New("job not found")
		},
	}

	svc := NewSparkJobsService(mockRepo)

	// Case 1: Job found (RUNNING -> running)
	status, err := svc.GetJobStatus(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "running" {
		t.Errorf("expected status 'running', got %q", status)
	}

	// Case 2: Job found (FINISHED -> completed)
	status, err = svc.GetJobStatus(context.Background(), "job-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "completed" {
		t.Errorf("expected status 'completed', got %q", status)
	}

	// Case 3: Job not found error propagation
	_, err = svc.GetJobStatus(context.Background(), "job-999")
	if err == nil {
		t.Error("expected error for unknown job, got nil")
	}
}
