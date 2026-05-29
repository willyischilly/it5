// Seed demo requests via HTTP API. Run: go run ./scripts/seed-demo
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const base = "http://localhost:8080"

type apiErr struct {
	status int
	body   string
}

func (e apiErr) Error() string { return fmt.Sprintf("HTTP %d: %s", e.status, e.body) }

func call(method, path string, body any, token string, out any) error {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, base+path, r)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return apiErr{status: resp.StatusCode, body: string(data)}
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

type loginResp struct {
	Token string `json:"token"`
}

type user struct {
	ID         uint   `json:"id"`
	Email      string `json:"email"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Patronymic string `json:"patronymic"`
}

type work struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type contour struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type task struct {
	ID       uint   `json:"id"`
	Work     *work  `json:"work"`
	Executor *user  `json:"executor"`
	Status   string `json:"status"`
}

type request struct {
	ID     uint    `json:"id"`
	Title  string  `json:"title"`
	Status string  `json:"status"`
	Tasks  []task  `json:"tasks"`
}

func ensureUser(email, password, last, first, patronymic, role string) (string, error) {
	var lr loginResp
	if err := call("POST", "/api/login", map[string]string{"email": email, "password": password}, "", &lr); err == nil {
		return lr.Token, nil
	}
	if err := call("POST", "/api/register", map[string]string{
		"email": email, "password": password,
		"last_name": last, "first_name": first, "patronymic": patronymic,
		"role": role,
	}, "", &lr); err != nil {
		return "", err
	}
	return lr.Token, nil
}

func main() {
	if err := call("GET", "/health", nil, "", nil); err != nil {
		fmt.Fprintln(os.Stderr, "Сервер не доступен на", base, "— запустите: go run ./cmd/main.go")
		os.Exit(1)
	}

	custTok, err := ensureUser("demo.customer@planner.local", "demo123456", "Сидоров", "Алексей", "Петрович", "customer")
	if err != nil {
		fatal(err)
	}
	exec1Tok, err := ensureUser("demo.executor1@planner.local", "demo123456", "Иванов", "Иван", "Иванович", "executor")
	if err != nil {
		fatal(err)
	}
	exec2Tok, err := ensureUser("demo.executor2@planner.local", "demo123456", "Петров", "Пётр", "Сергеевич", "executor")
	if err != nil {
		fatal(err)
	}

	var exec1, exec2 user
	must(call("GET", "/api/me", nil, exec1Tok, &exec1))
	must(call("GET", "/api/me", nil, exec2Tok, &exec2))

	var works []work
	var contours []contour
	must(call("GET", "/api/works", nil, custTok, &works))
	must(call("GET", "/api/contours", nil, custTok, &contours))

	workID := map[string]uint{}
	for _, w := range works {
		workID[w.Name] = w.ID
	}
	contourID := map[string]uint{}
	for _, c := range contours {
		contourID[c.Name] = c.ID
	}

	type taskSpec struct {
		work     string
		comment  string
		executor uint // 0 = none, 1 = exec1, 2 = exec2
	}

	create := func(title, contourName string, days int, specs []taskSpec) request {
		deadline := time.Now().AddDate(0, 0, days).UTC().Format(time.RFC3339)
		var req request
		must(call("POST", "/api/requests", map[string]any{
			"title": title, "contour_id": contourID[contourName], "deadline_at": deadline,
		}, custTok, &req))

		items := make([]map[string]any, 0, len(specs))
		for _, s := range specs {
			item := map[string]any{"work_id": workID[s.work], "comment": s.comment}
			switch s.executor {
			case 1:
				item["executor_id"] = exec1.ID
			case 2:
				item["executor_id"] = exec2.ID
			}
			items = append(items, item)
		}
		must(call("POST", fmt.Sprintf("/api/requests/%d/tasks", req.ID), map[string]any{"tasks": items}, custTok, &req))
		return req
	}

	submit := func(req request) request {
		var out request
		must(call("POST", fmt.Sprintf("/api/requests/%d/submit", req.ID), nil, custTok, &out))
		return out
	}

	fmt.Println("=== Seed demo requests ===")

	r1 := create("Миграция биллинга (подготовка)", "Dev", 14, []taskSpec{
		{"Развертывание VM", "VM для staging биллинга, 4 CPU / 16 GB RAM", 1},
		{"Настройка сети", "VLAN 120, доступ только из VPN", 1},
		{"Установка СУБД", "PostgreSQL 16, отдельный инстанс для миграции", 0},
	})
	fmt.Printf("[1] Draft #%d — %s (%d tasks, 1 без исполнителя)\n", r1.ID, r1.Title, len(r1.Tasks))

	r2 := create("Развёртывание CRM на Qa", "Qa", 10, []taskSpec{
		{"Развертывание приложения", "Backend CRM v2.4.1, blue-green", 1},
		{"Настройка мониторинга", "Grafana + Prometheus, алерт в Telegram", 2},
		{"Резервное копирование", "Ежедневный backup БД, retention 14 дней", 2},
		{"Приемочное тестирование", "Smoke + регресс по чек-листу заказчика", 1},
	})
	r2 = submit(r2)
	fmt.Printf("[2] Submitted #%d — %s (Иванов + Петров)\n", r2.ID, r2.Title)

	r3 := create("Обновление мониторинга Prod", "Prod", 7, []taskSpec{
		{"Настройка мониторинга", "Новые дашборды для API gateway", 2},
		{"Развертывание приложения", "Hotfix exporter v1.0.3", 2},
	})
	r3 = submit(r3)
	if len(r3.Tasks) > 0 {
		must(call("PUT", fmt.Sprintf("/api/tasks/%d/status", r3.Tasks[0].ID), map[string]string{"status": "in_progress"}, exec2Tok, nil))
	}
	must(call("GET", fmt.Sprintf("/api/requests/%d", r3.ID), nil, custTok, &r3))
	fmt.Printf("[3] In progress #%d — %s (status: %s)\n", r3.ID, r3.Title, r3.Status)

	fmt.Println()
	fmt.Println("Заказчик:  demo.customer@planner.local / demo123456")
	fmt.Println("Исполнитель 1: demo.executor1@planner.local / demo123456 (Иванов И.И.)")
	fmt.Println("Исполнитель 2: demo.executor2@planner.local / demo123456 (Петров П.С.)")
	fmt.Println("Swagger:", base+"/swagger")
}

func must(err error) {
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
