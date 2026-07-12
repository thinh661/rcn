package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/rcn/rcn/backend/internal/database"
)

// Mock SQL Driver for k8s_per_user_gateway tests
type k8sMockDriver struct{}

func (d *k8sMockDriver) Open(name string) (driver.Conn, error) {
	return &k8sMockConn{}, nil
}

type k8sMockConn struct{}

func (c *k8sMockConn) Prepare(query string) (driver.Stmt, error) {
	return &k8sMockStmt{query: query}, nil
}
func (c *k8sMockConn) Begin() (driver.Tx, error) { return &k8sMockTx{}, nil }
func (c *k8sMockConn) Close() error               { return nil }

type k8sMockTx struct{}

func (t *k8sMockTx) Commit() error   { return nil }
func (t *k8sMockTx) Rollback() error { return nil }

type k8sMockStmt struct {
	query string
}

func (s *k8sMockStmt) Close() error { return nil }
func (s *k8sMockStmt) NumInput() int {
	return -1
}
func (s *k8sMockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &k8sMockResult{}, nil
}

type k8sMockResult struct{}

func (r *k8sMockResult) LastInsertId() (int64, error) { return 1, nil }
func (r *k8sMockResult) RowsAffected() (int64, error) { return 1, nil }

var (
	dbMu          sync.Mutex
	dbPhase       = "spawning"
	dbMsg         = "Preparing kernel pod..."
	dbURL         = ""
	dbPodName     = ""
	dbCpuReq      = "500m"
	dbMemReq      = "1Gi"
	dbCpuLim      = "2000m"
	dbMemLim      = "4Gi"
	queryBehavior = "normal"
)

func (s *k8sMockStmt) Query(args []driver.Value) (driver.Rows, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if queryBehavior == "empty" {
		return &k8sMockRows{idx: 1}, nil // simulate no rows
	}

	return &k8sMockRows{
		columns: []string{"status", "phase_message", "pod_url", "pod_name", "cpu_request", "mem_request", "cpu_limit", "mem_limit"},
		rows: [][]driver.Value{
			{dbPhase, dbMsg, dbURL, dbPodName, dbCpuReq, dbMemReq, dbCpuLim, dbMemLim},
		},
		idx: 0,
	}, nil
}

type k8sMockRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *k8sMockRows) Columns() []string { return r.columns }
func (r *k8sMockRows) Close() error      { return nil }
func (r *k8sMockRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	for i, v := range r.rows[r.idx] {
		dest[i] = v
	}
	r.idx++
	return nil
}

var k8sRegisterOnce sync.Once

func setupGatewayMockDB(t *testing.T) {
	k8sRegisterOnce.Do(func() {
		sql.Register("k8smockdriver", &k8sMockDriver{})
	})
	db, err := sql.Open("k8smockdriver", "")
	if err != nil {
		t.Fatalf("Failed to open mock db: %v", err)
	}
	database.SetDB(db)
}

func TestPodCreation(t *testing.T) {
	setupGatewayMockDB(t)
	fakeClient := fake.NewSimpleClientset()

	cfg := K8sPerUserConfig{
		Namespace:     "test-ns",
		PodImage:      "test-image:latest",
		IdleTimeout:   10 * time.Minute,
		MaxPods:       5,
		CPURequest:    "200m",
		MemoryRequest: "500Mi",
		CPULimit:      "1000m",
		MemoryLimit:   "2Gi",
	}

	gw := &K8sPerUserGateway{
		cfg:        cfg,
		client:     fakeClient,
		touchBuf:   make(map[string]time.Time),
		stopCh:     make(chan struct{}),
		usageCache: make(map[string]cachedUsage),
	}

	userID := "user-alice"
	podName := podNameForUser(userID)

	dbMu.Lock()
	queryBehavior = "empty" // Return no row initially so EnsureSpawning knows it's a fresh spawn
	dbMu.Unlock()

	// EnsureSpawning should start the async spawn and return immediately
	err := gw.EnsureSpawning(userID, nil)
	if err != nil {
		t.Fatalf("EnsureSpawning failed: %v", err)
	}

	// Wait briefly for background goroutine to execute step 2 (create pod)
	time.Sleep(100 * time.Millisecond)

	// Verify pod was created in the fake client
	pod, err := fakeClient.CoreV1().Pods("test-ns").Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Pod was not created in K8s: %v", err)
	}

	// Verify pod spec details
	if pod.Spec.Containers[0].Image != "test-image:latest" {
		t.Errorf("Expected image 'test-image:latest', got '%s'", pod.Spec.Containers[0].Image)
	}
	cpuReq := pod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
	if cpuReq.String() != "200m" {
		t.Errorf("Expected CPU request '200m', got '%s'", cpuReq.String())
	}
}

func TestPodDeletion(t *testing.T) {
	setupGatewayMockDB(t)
	fakeClient := fake.NewSimpleClientset()

	userID := "user-bob"
	podName := podNameForUser(userID)

	// Pre-create the pod in fake client
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: "test-ns",
		},
	}
	_, _ = fakeClient.CoreV1().Pods("test-ns").Create(context.Background(), pod, metav1.CreateOptions{})

	gw := &K8sPerUserGateway{
		cfg: K8sPerUserConfig{
			Namespace: "test-ns",
		},
		client:     fakeClient,
		touchBuf:   make(map[string]time.Time),
		stopCh:     make(chan struct{}),
		usageCache: make(map[string]cachedUsage),
	}

	// Invoke Destroy
	err := gw.Destroy(userID)
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	// Verify pod is deleted (or deletion timestamp is set)
	_, err = fakeClient.CoreV1().Pods("test-ns").Get(context.Background(), podName, metav1.GetOptions{})
	if err == nil {
		t.Error("Expected pod to be deleted from K8s, but it still exists")
	}
}

func TestPodStatusPolling(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	// Pod starting but not ready yet
	p1 := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ContainerCreating",
							Message: "creating container",
						},
					},
					Ready: false,
				},
			},
		},
	}
	phase, msg := derivePhase(p1)
	if phase != PhasePulling {
		t.Errorf("Expected phase 'pulling', got '%s'", phase)
	}

	// Pod ready
	p2 := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.5",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
				},
			},
		},
	}
	url := readyURL(p2)
	if url != "http://10.0.0.5:8888" {
		t.Errorf("Expected readyURL 'http://10.0.0.5:8888', got '%s'", url)
	}
}

func TestErrorHandling(t *testing.T) {
	setupGatewayMockDB(t)

	// Create a fake client that returns errors on Get
	fakeClient := fake.NewSimpleClientset()
	fakeClient.PrependReactor("get", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("K8s API Server Unavailable")
	})

	gw := &K8sPerUserGateway{
		cfg: K8sPerUserConfig{
			Namespace: "test-ns",
		},
		client:     fakeClient,
		touchBuf:   make(map[string]time.Time),
		stopCh:     make(chan struct{}),
		usageCache: make(map[string]cachedUsage),
	}

	// Check pod health with broken K8s client, should fail gracefully returning false
	healthy := gw.podHealthy(context.Background(), "some-pod")
	if healthy {
		t.Error("Expected podHealthy to return false due to API errors, but got true")
	}
}
