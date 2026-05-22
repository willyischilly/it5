Схема и начальные данные применяются при старте API (GORM AutoMigrate + seed в internal/database/postgres.go).

Отдельный прогон SQL не нужен. Участник 3 может добавить init-скрипты в compose, если потребуется ИБ.

Таблицы: users, works, deployment_contours, requests, tasks, task_logs, request_logs.
