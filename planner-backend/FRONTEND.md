# Для фронтенда

Бэкенд крутится на http://localhost:8080 (порт в .env, по умолчанию 8080).

Нужны Go 1.22+ и PostgreSQL. Перед запуском скопируй .env.example в .env, укажи DB_PASSWORD и JWT_SECRET не короче 32 символов. Postgres должен работать.

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

Заказчика и исполнителя — через register с role customer или executor, либо admin создаёт пользователя POST /api/admin/users.

Что уже есть в базе

Четыре контура: Dev, Qa, Uat, Prod — GET /api/contours под токеном customer.
Семь работ в справочнике — GET /api/works (customer) или /api/admin/works (admin).

Кто куда ходит

Публично: register, login. С токеном: GET /api/me.

admin — /api/admin/users, works, contours, request-logs, task-logs
customer — /api/requests, works, contours, отчёт
executor — /api/tasks, смена статуса, GET /api/requests/:id если на заявке его задачи

Как обычно строить экраны

Заказчик логинится, берёт контуры и работы, создаёт заявку POST /api/requests (можно сразу deadline_at в ISO), правит черновик PUT /api/requests/:id.

Удаление заявки DELETE /api/requests/:id — в черновике (draft) или когда заявка отправлена (submitted) и все задачи ещё pending («в планах»).

Добавление работ POST /api/requests/:id/tasks — только в черновике. Тело tasks: [{ work_id, comment }] или work_ids: [1,2].

Удаление задачи DELETE /api/requests/:id/tasks/:task_id — в черновике или если у задачи статус pending (в планах), пока заявка не completed/overdue.

Отправка POST /api/requests/:id/submit. Если дедлайн прошёл — статус overdue, продление POST /api/requests/:id/extend-deadline с новым deadline_at.

Исполнитель видит GET /api/tasks, меняет статус PUT /api/tasks/:id/status: сначала in_progress, потом completed.

Заказчик смотрит GET /api/requests/:id и отчёты:

По одной заявке:
- JSON: GET /api/requests/:id/report
- PDF: GET /api/requests/:id/report/pdf

По всем заявкам заказчика (сводка: статусы, что выполнено, что нет):
- JSON: GET /api/requests/reports/summary
- PDF: GET /api/requests/reports/summary/pdf

Статусы заявки: draft, submitted, in_progress, completed, overdue.
Статусы задачи: pending, in_progress, completed. Перескакивать нельзя.

После submit задачи разъезжаются по исполнителям по кругу, если их несколько.

Ошибки приходят JSON с полем error. Коды как обычно: 400, 401, 403, 404.

Переменные .env

PORT — порт сервера
DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME — postgres
JWT_SECRET, JWT_EXPIRE_HOURS
CORS_ORIGINS — по желанию
SEED_ADMIN_EMAIL, SEED_ADMIN_PASSWORD — кто создаётся первым админом

Проверка с твоей машины: поднять сервер и выполнить scripts\full-checkup.ps1
