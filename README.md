# Oper Plan

Бұл репозиторийде:
- `backend` — Go (Gin + GORM + PostgreSQL)
- `frontend` — Vue 3 + Vite

## Quick Start

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
  -p 5433:5432 \
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

4. Ашылатын адрестер:
- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`

Толық нұсқаулықтар:
- `backend/README.md`
- `frontend/README.md`
