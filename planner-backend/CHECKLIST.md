Чеклист бэкенда

REST API Go, Gin, слои handlers / services / repositories

PostgreSQL, AutoMigrate, seed 7 работ и 4 контура

JWT, роли admin / customer / executor

Валидация полей и normative_hours >= 1

task_logs и request_logs, просмотр admin: request-logs, task-logs

OpenAPI, Swagger, Postman

CORS для фронта

Сценарий: admin → customer план (комментарии к задачам, дедлайн) → submit → executor → overdue/продление → отчёт PDF

PDF: internal/services/pdffonts/Arial.ttf (кириллица)

Контуры: любые названия (до 50 символов), admin CRUD

Проверка: scripts/full-checkup.ps1 при запущенном сервере

Не входит в бэкенд: UI, docker-compose
