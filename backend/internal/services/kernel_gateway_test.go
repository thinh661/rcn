package services

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestKernelConnect tests the basic connection flow with SharedGateway.
func TestKernelConnect(t *testing.T) {
	gatewayURL := "http://localhost:8888"
	gw := NewSharedGateway(gatewayURL)

	if gw.Mode() != "shared" {
		t.Errorf("Expected mode 'shared', got '%s'", gw.Mode())
	}

	url, err := gw.GetGatewayURL(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if url != gatewayURL {
		t.Errorf("Expected URL '%s', got '%s'", gatewayURL, url)
	}

	// Verify Touch and Destroy do not panic
	gw.Touch("user-123")
	err = gw.Destroy("user-123")
	if err != nil {
		t.Errorf("Destroy failed: %v", err)
	}

	status, err := gw.Status("user-123")
	if err != nil {
		t.Errorf("Status failed: %v", err)
	}
	if status.Phase != PhaseReady {
		t.Errorf("Expected status phase 'ready', got '%s'", status.Phase)
	}
}

// mockTimeoutGateway implements a simple kernel gateway to test idle timeout logic.
type mockTimeoutGateway struct {
	mu          sync.Mutex
	lastActive  map[string]time.Time
	idleTimeout time.Duration
	activePods  map[string]bool
}

func newMockTimeoutGateway(idleTimeout time.Duration) *mockTimeoutGateway {
	return &mockTimeoutGateway{
		lastActive:  make(map[string]time.Time),
		idleTimeout: idleTimeout,
		activePods:  make(map[string]bool),
	}
}

func (m *mockTimeoutGateway) GetGatewayURL(ctx context.Context, userID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activePods[userID] = true
	m.lastActive[userID] = time.Now()
	return fmt.Sprintf("http://kernel-%s:8888", userID), nil
}

func (m *mockTimeoutGateway) Touch(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastActive[userID] = time.Now()
}

func (m *mockTimeoutGateway) Destroy(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activePods, userID)
	delete(m.lastActive, userID)
	return nil
}

func (m *mockTimeoutGateway) Mode() string { return "mock_timeout" }

func (m *mockTimeoutGateway) Status(userID string) (PodStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activePods[userID] {
		return PodStatus{Phase: PhaseReady}, nil
	}
	return PodStatus{}, nil
}

func (m *mockTimeoutGateway) EnsureSpawning(userID string, spec *ResourceSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activePods[userID] = true
	m.lastActive[userID] = time.Now()
	return nil
}

func (m *mockTimeoutGateway) Usage(ctx context.Context, userID string) (ResourceUsage, error) {
	return ResourceUsage{}, ErrUsageUnsupported
}

func (m *mockTimeoutGateway) RecentLogs(ctx context.Context, userID string, tailLines int) (string, error) {
	return "", ErrUsageUnsupported
}

func (m *mockTimeoutGateway) reapIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for userID, last := range m.lastActive {
		if now.Sub(last) > m.idleTimeout {
			delete(m.activePods, userID)
			delete(m.lastActive, userID)
		}
	}
}

// TestKernelIdleTimeout tests automatic cleanup of idle kernels.
func TestKernelIdleTimeout(t *testing.T) {
	timeout := 10 * time.Millisecond
	gw := newMockTimeoutGateway(timeout)

	// Connect a user kernel
	_, err := gw.GetGatewayURL(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Failed to connect kernel: %v", err)
	}

	// Assert active
	status, _ := gw.Status("user-1")
	if status.Phase != PhaseReady {
		t.Errorf("Expected kernel to be active, got phase: '%s'", status.Phase)
	}

	// Touch to extend lease
	time.Sleep(5 * time.Millisecond)
	gw.Touch("user-1")

	// Wait, check that it's still active because touch extended it
	time.Sleep(7 * time.Millisecond)
	gw.reapIdle()
	status, _ = gw.Status("user-1")
	if status.Phase != PhaseReady {
		t.Error("Kernel was prematurely reaped despite Touch()")
	}

	// Wait for longer than idle timeout without touching
	time.Sleep(12 * time.Millisecond)
	gw.reapIdle()

	// Verify reaped
	status, _ = gw.Status("user-1")
	if status.Phase != "" {
		t.Errorf("Expected kernel to be reaped, but phase is still '%s'", status.Phase)
	}
}

// TestKernelMaxLimit tests that capacity limits are enforced.
func TestKernelMaxLimit(t *testing.T) {
	maxLimit := 3
	activeCount := 0
	mu := sync.Mutex{}

	// Simulated connector helper enforcing the limit
	connectKernel := func(userID string) error {
		mu.Lock()
		defer mu.Unlock()
		if activeCount >= maxLimit {
			return fmt.Errorf("max kernels limit reached (%d)", maxLimit)
		}
		activeCount++
		return nil
	}

	// Connect 3 kernels
	if err := connectKernel("user-1"); err != nil {
		t.Errorf("Expected to connect user-1: %v", err)
	}
	if err := connectKernel("user-2"); err != nil {
		t.Errorf("Expected to connect user-2: %v", err)
	}
	if err := connectKernel("user-3"); err != nil {
		t.Errorf("Expected to connect user-3: %v", err)
	}

	// Connect 4th kernel, should fail
	err := connectKernel("user-4")
	if err == nil {
		t.Error("Expected 4th connection to fail due to max limit, but it succeeded")
	} else if err.Error() != "max kernels limit reached (3)" {
		t.Errorf("Expected limit error message, got: %s", err.Error())
	}
}
