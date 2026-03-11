# Oper Plan Backend (Gin)

## Modules

```text
cmd/api
internal/
  auth/
  user/
  plan/
  task/
  report/
  files/
  middleware/
  config/
  handler/
  server/
  pkg/logger/
```

## Implemented API Skeleton

- `POST /login`
- `POST /logout`
- `POST /register`
- `GET /me`
- `POST /change-password`
- `GET /users` (admin)
- `POST /users` (admin)
- `GET /plans`
- `POST /plans`
- `GET /plan-records/:id`
- `PATCH /plan-records/:id/status`
- `GET /plans/years`
- `GET /plans/indicators?year=2026`
- `PUT /plans/indicators/:indicator_id?year=2026` (admin)
- `GET /tasks`
- `POST /tasks`
- `GET /tasks/:id`
- `PATCH /tasks/:id`
- `PATCH /tasks/:id/status`
- `DELETE /tasks/:id`
- `POST /tasks/:id/report`
- `GET /tasks/:id/reports`
- `GET /planning-period`
- `POST /planning-period` (admin)
- `PATCH /planning-period/:id` (admin)

## Run

1. PostgreSQL контейнері жұмыс істеп тұрғанын тексеріңіз:

```bash
docker ps -a
docker start postgres-db-op
```

2. `backend/.env` ішінде DB параметрлері дұрыс екенін тексеріңіз:

```env
PORT=8080
LOG_LEVEL=DEBUG
DB_HOST=localhost
DB_PORT=5433
DB_USER=admin
DB_PASSWORD=admin123
DB_NAME=oper-plan
DB_SSLMODE=disable
DB_TIMEZONE=Asia/Qyzylorda
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173,http://localhost:4173,http://127.0.0.1:4173
SESSION_TTL_HOURS=24
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_PASSWORD=
```

3. Backend-ті іске қосыңыз:

```bash
cd backend
go run ./cmd/api
```

Если нужно автоматически создать первый `admin`, задайте `BOOTSTRAP_ADMIN_PASSWORD` перед запуском. Без этого backend больше не создает администратора с предсказуемым паролем.

4. Тексеру:
- API health: `http://localhost:8080/healthz`
- Swagger UI: `http://localhost:8080/swagger/index.html`

## Test Users Seed (Console)

Тест үшін 4 қолданушыны (1 admin, 1 viewer, 2 prorector) бір командамен құру:

```bash
cat sql/seed_test_users.sql | docker exec -i postgres-db-op psql -U admin -d oper-plan
```

Логин/пароль:

| Role | Username | Password |
| --- | --- | --- |
| admin | `admin_test` | `AdminTest1` |
| viewer | `viewer_test` | `ViewerTest1` |
| prorector | `prorector_aitimov` | `ProrectorA1` |
| prorector | `prorector_toktarov` | `ProrectorB1` |

## PostgreSQL (Docker)

Backend now connects to PostgreSQL on startup and runs `AutoMigrate` for:
- `users`
- `plans`
- `tasks`
- `reports`
- `task_logs`
- `planning_period_indicators`

Additionally, backend now runs explicit SQL migrations (`CREATE TABLE IF NOT EXISTS ...`) at startup for all core tables. This guarantees required tables exist even if previous migrations were skipped.

Default `.env` values are aligned with your Docker command:

```env
DB_HOST=localhost
DB_PORT=5433
DB_USER=admin
DB_PASSWORD=admin123
DB_NAME=oper-plan
DB_SSLMODE=disable
DB_TIMEZONE=Asia/Qyzylorda
```

## Swagger

- UI: `http://localhost:8080/swagger/index.html`
- OpenAPI files: `backend/docs/swagger.yaml` and `backend/docs/swagger.json`

Regenerate docs after changing handlers/annotations:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g cmd/api/main.go -o docs --parseInternal
```

## Notes

`plans`, `tasks`, `reports`, and `planning-period` endpoints now use explicit SQL queries (`SELECT/INSERT/UPDATE/DELETE`) against PostgreSQL.
