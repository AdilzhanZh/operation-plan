# Oper Plan

Бұл репозиторийде:
- `backend` — Go (Gin + GORM + PostgreSQL)
- `frontend` — Vue 3 + Vite

## Docker Run

Жоба толықтай Docker арқылы іске қосылады: PostgreSQL, backend және frontend бір `docker compose` стегімен көтеріледі.

### 1. Қажет болса env дайындау

```bash
cp .env.example .env
```

Әдепкі параметрлер:
- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`
- PostgreSQL: `localhost:5432`

### 2. Барлығын іске қосу

```bash
docker compose up --build -d
```

### 3. Логтарды тексеру

```bash
docker compose logs -f
```

### 4. Тоқтату

```bash
docker compose down
```

Егер PostgreSQL volume-мен бірге толық тазалау керек болса:

```bash
docker compose down -v
```

## Docker Architecture

- `postgres` — `postgres:17-alpine`
- `backend` — multi-stage Go build, контейнер ішінде `:8080`
- `frontend` — multi-stage Vite build + Nginx
- `frontend` контейнері `/api` сұраныстарын `backend:8080` адресіне proxy арқылы жібереді
- Жүктелген отчет файлдары host-та `backend/storage` ішінде сақталады

## Default Login

Backend бірінші іске қосылған кезде әдепкі админ қолданушы автоматты түрде жасалады:

| Role | Username | Password |
| --- | --- | --- |
| admin | `admin` | `Admin123` |

Қосымша тест қолданушыларды қажет етсеңіз, backend пен PostgreSQL іске қосылғаннан кейін seed орындауға болады:

```bash
docker compose exec -T postgres psql -U admin -d oper-plan < backend/sql/seed_test_users.sql
```

Seed логиндері:

| Role | Username | Password |
| --- | --- | --- |
| admin | `admin_test` | `AdminTest1` |
| viewer | `viewer_test` | `ViewerTest1` |
| prorector | `prorector_aitimov` | `ProrectorA1` |
| prorector | `prorector_toktarov` | `ProrectorB1` |

## Health Checks

- Backend health: `http://localhost:8080/healthz`
- Frontend entrypoint: `http://localhost:5173`
- Swagger UI: `http://localhost:8080/swagger/index.html`

## Local Run Without Docker

1. PostgreSQL контейнерін іске қосыңыз (егер бұрын құрылған болса — `docker start`):

```bash
docker start postgres-db-op
```

Егер контейнер әлі құрылмаған болса:

```bash
docker run --name postgres-db-op \
  -e POSTGRES_USER=admin \
  -e POSTGRES_PASSWORD=admin123 \
  -e POSTGRES_DB=oper-plan \
  -p 5432:5432 \
  -v gdata:/var/lib/postgresql/data \
  -d postgres:17
```

2. Backend іске қосу:

```bash
cd backend
go run ./cmd/api
```

3. Frontend іске қосу (жаңа терминалда):

```bash
cd frontend
npm install
npm run dev
```

Толық нұсқаулықтар:
- `backend/README.md`
- `frontend/README.md`
