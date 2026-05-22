# Planner Backend

API для планирования развёртывания: заявки, задачи, справочник работ, контуры, отчёты. Стек: Go, Gin, PostgreSQL, JWT.

Инструкция для фронта — FRONTEND.md. Чеклист сдачи — CHECKLIST.md.

Запуск

Скопируй .env.example в .env и пропиши пароль PostgreSQL и JWT_SECRET (минимум 32 символа). PostgreSQL должен быть запущен.

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

База planner и таблицы создаются при первом запуске. Подтягиваются контуры Dev/Qa/Uat/Prod, семь работ в справочнике и admin, если его ещё нет.

Роли

admin — /api/admin/... и журналы GET /api/admin/request-logs, /api/admin/task-logs
customer — заявки, works, contours, отчёт
executor — /api/tasks и просмотр заявки, если есть его задачи

В запросах: Authorization: Bearer и токен после логина.

Структура каталогов: cmd — вход, internal — логика, pkg — общие утилиты, postman — коллекция, scripts — сборка и проверки. OpenAPI лежит в internal/app/static и отдаётся с сервера.
