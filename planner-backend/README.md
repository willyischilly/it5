# Planner Backend

API для планирования развёртывания: заявки, задачи, справочник работ, контуры, отчёты. Стек: Go, Gin, PostgreSQL, JWT.

Инструкция для фронта — FRONTEND.md. Чеклист сдачи — CHECKLIST.md.

Запуск

Скопируй .env.example в .env (в примере `DB_PASSWORD=102030` для локального Postgres) и при необходимости поправь JWT_SECRET (минимум 32 символа). PostgreSQL должен быть запущен.

```
Copy-Item .env.example .env
go run ./cmd/main.go
```

Можно через скрипты: scripts\build.ps1, потом scripts\run.ps1. Если exe нет, run.ps1 сам вызовет go run.

После старта:

http://localhost:8080/health
http://localhost:8080/swagger
http://localhost:8080/api/openapi.yaml

Проверить API целиком (сервер уже должен работать): scripts\full-checkup.ps1

Демо-заявки и тестовые пользователи (сервер + Postgres должны быть запущены):

```
go run ./scripts/seed-demo
```

Создаёт demo.customer@planner.local, двух исполнителей и три заявки. Пароль: demo123456.

Требования: Go 1.21+, PostgreSQL (порт 5432). Перед запуском убедитесь, что служба Postgres работает (например postgresql-x64-18).

База planner и таблицы создаются при первом запуске. Подтягиваются контуры Dev/Qa/Uat/Prod, семь работ в справочнике и admin, если его ещё нет.

Роли

admin — /api/admin/... и журналы GET /api/admin/request-logs, /api/admin/task-logs
customer — заявки, works, contours, GET /api/executors, назначение исполнителей на задачи, отчёт JSON/PDF (GET /api/requests/:id/report/pdf)
executor — /api/requests, /api/contours, /api/tasks, смена статуса своих назначенных задач

В запросах: Authorization: Bearer и токен после логина.

Структура каталогов: cmd — вход, internal — логика, pkg — общие утилиты, postman — коллекция, scripts — сборка и проверки. OpenAPI лежит в internal/app/static и отдаётся с сервера.
