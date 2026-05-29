# Для фронтенда

Бэкенд крутится на http://localhost:8080 (порт в .env, по умолчанию 8080).

Нужны Go 1.22+ и PostgreSQL. Перед запуском скопируй .env.example в .env (пароль БД по умолчанию `102030`) и JWT_SECRET не короче 32 символов. Postgres должен работать.

Запуск:

```
cd planner-backend
Copy-Item .env.example .env
go run ./cmd/main.go
```

Или scripts\run.ps1 — там же остановится старый процесс на 8080, если висит.

Жив ли сервер: GET /health, ответ {"status":"ok"}.

Документация

Swagger: http://localhost:8080/swagger
Спека: http://localhost:8080/api/openapi.yaml
Postman: postman/planner-api.postman_collection.json

Логин и токен

POST /api/login или /api/register — в ответе token и user (с полем role). После перезагрузки страницы: GET /api/me с тем же токеном.

Заголовок:

Authorization: Bearer <token>
Content-Type: application/json

CORS уже включён для локальной разработки. Если фронт на другом порту и браузер ругается, в .env можно добавить:

CORS_ORIGINS=http://localhost:5173,http://localhost:3000

Если строку не задавать, origin не режется.

Тестовый admin (создаётся при первом старте, если в базе пусто):

admin@planner.local
admin123456

Заказчика и исполнителя — через register с role customer или executor и полями last_name, first_name, patronymic (фамилия, имя, отчество), либо admin создаёт пользователя POST /api/admin/users с теми же полями.

Что уже есть в базе

Четыре контура: Dev, Qa, Uat, Prod — GET /api/contours (customer и executor). У каждого контура есть description; админ задаёт при POST/PUT /api/admin/contours.
Семь работ в справочнике — GET /api/works (customer) или /api/admin/works (admin).

Кто куда ходит

Публично: register, login. С токеном: GET /api/me.

admin — /api/admin/users, works (название, **description** — описание работы в справочнике, нормочасы), contours (name, **description**), request-logs, task-logs. Описание задачи в заявке = `work.description` из справочника: PUT /api/admin/works/:id с полем `description`.
customer — /api/requests, works, contours, GET /api/executors, назначение исполнителей, отчёт
executor — GET /api/requests (все заявки с задачами), GET /api/contours, GET /api/tasks (все задачи), PUT /api/tasks/:id/status (только свои назначенные), GET /api/requests/:id

Как обычно строить экраны

Заказчик логинится, берёт контуры и работы, создаёт заявку POST /api/requests (deadline_at опционален, ISO, срок всей заявки), правит черновик PUT /api/requests/:id (deadline_at или clear_deadline true чтобы убрать срок). У задач дедлайна нет; просрочка (overdue) — только у заявки с заданным дедлайном.

Удаление заявки DELETE /api/requests/:id — в черновике (draft) или когда заявка отправлена (submitted) и все задачи ещё pending («в планах»).

Добавление работ POST /api/requests/:id/tasks — только в черновике. Тело tasks: [{ work_id, comment, executor_id? }] или work_ids: [1,2]. Исполнителя можно указать сразу или позже.

Назначение исполнителей (только черновик):
- PUT /api/requests/:id/tasks/:task_id/assign — тело { executor_id }
- PUT /api/requests/:id/tasks/assign — тело { assignments: [{ task_id, executor_id }, ...] }
- GET /api/executors — список исполнителей для выбора

Удаление задачи DELETE /api/requests/:id/tasks/:task_id — в черновике или если у задачи статус pending (в планах), пока заявка не completed/overdue.

Отправка POST /api/requests/:id/submit — только если у каждой задачи назначен executor_id. Если дедлайн прошёл — статус overdue, продление POST /api/requests/:id/extend-deadline с новым deadline_at.

Исполнитель видит все заявки GET /api/requests, контуры GET /api/contours и все задачи GET /api/tasks. После submit заказчиком задачи уже назначены исполнителям (в ответе у задачи executor с full_name). PUT /api/tasks/:id/status: in_progress, затем completed (только по своим задачам).

Заказчик смотрит GET /api/requests/:id и отчёты:

По одной заявке:
- JSON: GET /api/requests/:id/report
- PDF: GET /api/requests/:id/report/pdf

По всем заявкам заказчика (сводка: статусы, что выполнено, что нет):
- JSON: GET /api/requests/reports/summary
- PDF: GET /api/requests/reports/summary/pdf

Статусы заявки: draft, submitted, in_progress, completed, overdue.
Статусы задачи: pending, in_progress, completed. Перескакивать нельзя.

Ошибки приходят JSON с полем error. Коды как обычно: 400, 401, 403, 404.

Переменные .env

PORT — порт сервера
DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME — postgres
JWT_SECRET, JWT_EXPIRE_HOURS
CORS_ORIGINS — по желанию
SEED_ADMIN_EMAIL, SEED_ADMIN_PASSWORD — кто создаётся первым админом

Проверка с твоей машины: поднять сервер и выполнить scripts\full-checkup.ps1
