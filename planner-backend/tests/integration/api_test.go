package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"planner-backend/internal/app"
	"planner-backend/internal/config"
	"planner-backend/internal/database"
	"planner-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var (
	testDB     *gorm.DB
	testServer *httptest.Server
	testCfg    *config.Config
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	_ = os.Chdir("../..")
	_ = godotenv.Load(".env")

	testCfg = &config.Config{
		Port:           "8080",
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "102030"),
		DBName:         getEnv("DB_NAME", "planner"),
		JWTSecret:      "integration_test_secret_key_32chars",
		JWTExpireHours: 24,
	}

	var err error
	testDB, err = database.Connect(testCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP integration: database: %v\n", err)
		os.Exit(0)
	}

	// Make tests repeatable: keep seeded works/contours/admin, wipe mutable data.
	// Without this, old executors/customers remain and task assignment becomes non-deterministic.
	_ = testDB.Exec("TRUNCATE TABLE task_logs, request_logs, tasks, requests RESTART IDENTITY CASCADE").Error
	_ = testDB.Where("email <> ?", "admin@planner.local").Delete(&models.User{}).Error

	r, _ := app.NewRouter(testCfg, testDB)
	testServer = httptest.NewServer(r)
	code := m.Run()
	testServer.Close()
	os.Exit(code)
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func api(method, path string, body interface{}, token string) (*http.Response, []byte) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, testServer.URL+path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, data
}

func login(t *testing.T, email, password string) string {
	t.Helper()
	resp, data := api("POST", "/api/login", map[string]string{"email": email, "password": password}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login %s: %d %s", email, resp.StatusCode, data)
	}
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(data, &out)
	if out.Token == "" {
		t.Fatalf("empty token for %s", email)
	}
	return out.Token
}

func TestHealth(t *testing.T) {
	resp, data := api("GET", "/health", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: %d %s", resp.StatusCode, data)
	}
}

func TestFullCaseScenario(t *testing.T) {
	adminToken := login(t, "admin@planner.local", "admin123456")

	resp, data := api("GET", "/api/admin/works", nil, adminToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin works: %d %s", resp.StatusCode, data)
	}
	var works []models.Work
	_ = json.Unmarshal(data, &works)
	if len(works) < 5 {
		t.Fatalf("expected >=5 seeded works, got %d", len(works))
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	custEmail := "cust_" + suffix + "@test.local"
	execEmail := "exec_" + suffix + "@test.local"

	resp, data = api("POST", "/api/register", map[string]string{
		"email": custEmail, "password": "123456", "name": "Test Customer", "role": "customer",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register customer: %d %s", resp.StatusCode, data)
	}

	resp, data = api("POST", "/api/register", map[string]string{
		"email": execEmail, "password": "123456", "name": "Test Executor", "role": "executor",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register executor: %d %s", resp.StatusCode, data)
	}

	custToken := login(t, custEmail, "123456")
	execToken := login(t, execEmail, "123456")

	resp, data = api("GET", "/api/contours", nil, custToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("contours: %d %s", resp.StatusCode, data)
	}
	var contours []models.DeploymentContour
	_ = json.Unmarshal(data, &contours)
	if len(contours) < 4 {
		t.Fatalf("expected 4 contours, got %d", len(contours))
	}

	resp, data = api("POST", "/api/requests", map[string]interface{}{
		"title": "Test deployment", "contour_id": contours[0].ID,
	}, custToken)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create request: %d %s", resp.StatusCode, data)
	}
	var req models.Request
	_ = json.Unmarshal(data, &req)

	resp, data = api("POST", fmt.Sprintf("/api/requests/%d/tasks", req.ID), map[string][]uint{
		"work_ids": {works[0].ID, works[1].ID},
	}, custToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add tasks: %d %s", resp.StatusCode, data)
	}

	resp, data = api("POST", fmt.Sprintf("/api/requests/%d/tasks", req.ID), map[string][]uint{
		"work_ids": {works[0].ID},
	}, custToken)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate work should fail, got %d %s", resp.StatusCode, data)
	}

	resp, data = api("POST", fmt.Sprintf("/api/requests/%d/submit", req.ID), nil, custToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: %d %s", resp.StatusCode, data)
	}
	_ = json.Unmarshal(data, &req)
	if req.Status != models.RequestStatusSubmitted {
		t.Fatalf("expected submitted, got %s", req.Status)
	}

	var reqLogs int64
	testDB.Model(&models.RequestLog{}).Where("request_id = ?", req.ID).Count(&reqLogs)
	if reqLogs < 2 {
		t.Fatalf("expected >=2 request_logs, got %d", reqLogs)
	}

	resp, data = api("GET", "/api/tasks", nil, execToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("executor tasks: %d %s", resp.StatusCode, data)
	}
	var tasks []models.Task
	_ = json.Unmarshal(data, &tasks)
	if len(tasks) < 2 {
		t.Fatalf("executor should have >=2 tasks, got %d", len(tasks))
	}

	taskID := tasks[0].ID
	resp, data = api("PUT", fmt.Sprintf("/api/tasks/%d/status", taskID), map[string]string{
		"status": "in_progress",
	}, execToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("in_progress: %d %s", resp.StatusCode, data)
	}

	var taskLogs int64
	testDB.Model(&models.TaskLog{}).Where("task_id = ?", taskID).Count(&taskLogs)
	if taskLogs != 1 {
		t.Fatalf("expected 1 task_log, got %d", taskLogs)
	}

	resp, data = api("PUT", fmt.Sprintf("/api/tasks/%d/status", taskID), map[string]string{
		"status": "completed",
	}, execToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("completed: %d %s", resp.StatusCode, data)
	}

	for _, tid := range []uint{tasks[0].ID, tasks[1].ID} {
		if tid == taskID {
			continue
		}
		api("PUT", fmt.Sprintf("/api/tasks/%d/status", tid), map[string]string{"status": "in_progress"}, execToken)
		api("PUT", fmt.Sprintf("/api/tasks/%d/status", tid), map[string]string{"status": "completed"}, execToken)
	}

	resp, data = api("GET", fmt.Sprintf("/api/requests/%d", req.ID), nil, custToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("customer view request: %d %s", resp.StatusCode, data)
	}
	_ = json.Unmarshal(data, &req)
	if req.Status != models.RequestStatusCompleted {
		t.Fatalf("expected completed request, got %s", req.Status)
	}

	resp, data = api("GET", fmt.Sprintf("/api/requests/%d/report?format=json", req.ID), nil, custToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("report json: %d %s", resp.StatusCode, data)
	}

	resp, data = api("GET", fmt.Sprintf("/api/requests/%d/report?format=pdf", req.ID), nil, custToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("report pdf: %d %s", resp.StatusCode, data)
	}
	if resp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("expected application/pdf, got %s", resp.Header.Get("Content-Type"))
	}
	if len(data) < 100 {
		t.Fatalf("pdf too small: %d bytes", len(data))
	}
}

func TestValidationAndAuth(t *testing.T) {
	resp, _ := api("GET", "/api/admin/users", nil, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: expected 401, got %d", resp.StatusCode)
	}

	resp, data := api("POST", "/api/register", map[string]string{
		"email": "bad", "password": "123456", "name": "X", "role": "customer",
	}, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad email: expected 400, got %d %s", resp.StatusCode, data)
	}

	resp, data = api("POST", "/api/admin/works", map[string]interface{}{
		"name": "Bad", "normative_hours": -1,
	}, login(t, "admin@planner.local", "admin123456"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("negative hours: expected 400, got %d %s", resp.StatusCode, data)
	}
}
