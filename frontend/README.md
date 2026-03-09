# Oper Plan Frontend (Vue 3 + Vite)

## Stack

- Vue 3 (Composition API)
- Vue Router
- Pinia
- Axios

## Directory Structure

```text
src/
  pages/
  components/
  services/
  router/
  store/
```

## Scripts

1. Алдымен backend іске қосылғанына көз жеткізіңіз (`http://localhost:8080`).

2. Frontend-ті іске қосыңыз:

```bash
cd frontend
npm install
npm run dev
```

3. Production build:

```bash
npm run build
npm run preview
```

## API Base URL

By default frontend uses `/api` and Vite proxy forwards requests to `http://127.0.0.1:8080`.
This removes CORS issues during local development.

Example:

```env
# Optional: override axios base URL (for example in production)
VITE_API_BASE_URL=/api

# Optional: override Vite proxy target in development
VITE_API_PROXY_TARGET=http://127.0.0.1:8080
```
