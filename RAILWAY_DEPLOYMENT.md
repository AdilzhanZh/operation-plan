# Railway-ге жобаны орналастыру нұсқаулығы / Railway Deployment Guide

Бұл нұсқаулық бүкіл жобаны (PostgreSQL, Go Backend, Vue Frontend) Railway платформасына қалай орналастыру керектігін сипаттайды.

This guide describes how to deploy the entire project (PostgreSQL, Go Backend, Vue Frontend) to the Railway platform.

---

## 1. Railway-де жоба құру / Create a Project in Railway

1. Railway-ге кіріп, **New Project** батырмасын басыңыз.
2. **Provision PostgreSQL** таңдап, деректер қорын қосыңыз.

1. Go to Railway and click **New Project**.
2. Select **Provision PostgreSQL** to add a database.

---

## 2. Backend орналастыру / Deploy Backend

1. **New** -> **GitHub Repo** таңдап, өз репозиториіңізді қосыңыз.
2. Жасалған сервистің **Settings** қойындысына өтіңіз:
   - **Root Directory** параметрін `/backend` деп орнатыңыз. (Railway автоматты түрде `/backend/Dockerfile` арқылы жинайды).
3. **Variables** қойындысына өтіп, келесі айнымалыларды қосыңыз:
   - `DATABASE_URL`: `${{Postgres.DATABASE_URL}}` (Немесе сіздің PostgreSQL сервисіңіздің сілтемесі. Railway мұны автоматты түрде ұсына алады).
   - `PORT`: `8080`
   - `CORS_ALLOWED_ORIGINS`: `https://ваша-фронтенд-сілтеме.up.railway.app` (Frontend дайын болған кезде оның сілтемесін осында көрсету керек).
   - `BOOTSTRAP_ADMIN_PASSWORD`: `СіздіңӘкімшіПароліңіз` (Әдепкі `admin` қолданушысы үшін).
   - `LOG_LEVEL`: `INFO`

1. Select **New** -> **GitHub Repo** and choose your repository.
2. Go to the **Settings** tab of the newly created service:
   - Set **Root Directory** to `/backend`. (Railway will automatically build using `/backend/Dockerfile`).
3. Go to the **Variables** tab and add the following:
   - `DATABASE_URL`: `${{Postgres.DATABASE_URL}}` (Or reference your PostgreSQL service variable. Railway can auto-inject this).
   - `PORT`: `8080`
   - `CORS_ALLOWED_ORIGINS`: `https://your-frontend-domain.up.railway.app` (Once frontend is deployed, update this with the frontend domain).
   - `BOOTSTRAP_ADMIN_PASSWORD`: `YourAdminPassword` (For the default `admin` user).
   - `LOG_LEVEL`: `INFO`

---

## 3. Frontend орналастыру / Deploy Frontend

1. **New** -> **GitHub Repo** таңдап, тағы да сол репозиториіңізді қосыңыз.
2. Жасалған сервистің **Settings** қойындысына өтіңіз:
   - Сервис атауын (Service Name) `frontend` деп өзгертіңіз.
   - **Root Directory** параметрін `/frontend` деп орнатыңыз. (Railway автоматты түрде `/frontend/Dockerfile` арқылы жинайды).
3. **Variables** қойындысына өтіп, келесі айнымалыларды қосыңыз:
   - `BACKEND_URL`: `http://backend.railway.internal:8080` (Егер сіздің backend сервисіңіз Railway-де `backend` деп аталса, осы сілтеме арқылы ішкі желіде байланысады).
4. **Settings** қойындысында **Generate Domain** батырмасын басып, сыртқы сілтеме (Public URL) алыңыз.

1. Select **New** -> **GitHub Repo** and connect the same repository again.
2. Go to the **Settings** tab of this new service:
   - Rename the service to `frontend`.
   - Set **Root Directory** to `/frontend`. (Railway will build using `/frontend/Dockerfile`).
3. Go to the **Variables** tab and add:
   - `BACKEND_URL`: `http://backend.railway.internal:8080` (Assuming your backend service is named `backend` in Railway, this enables internal network communication).
4. Go to **Settings** and click **Generate Domain** to get a public URL for your frontend.

2. **Для фронтенда (если там Nginx / Node.js):** Если вы используете собственный `Dockerfile` с Nginx для раздачи статики, Nginx должен слушать порт, который Railway передает внутрь контейнера. 
3. **Настройки в Railway:** Зайдите в карточку упавшего сервиса -> **Settings** -> раздел **Networking**. Найдите поле **Port**. Там должен быть указан тот порт, который реально слушает процесс внутри контейнера. Если ваш Go-бэк или Nginx жестко настроен на `8080`, впишите туда `8080` вручную, чтобы Railway знал, куда перенаправлять трафик.

---

### Шаг 2: Смотрим живые Deploy Logs
Чтобы не гадать, какая именно папка (фронтенд или бэкенд) выдает ошибку со скриншота `image_8c4598.png`:

1. Вернитесь в панель Railway.
2. Кликните по карточке сервиса, ссылку которого вы пытались открыть.
3. Перейдите во вкладку **Deploy Logs**.
4. Обновите страницу с ошибкой `image_8c4598.png` в соседней вкладке, чтобы спровоцировать новый запрос, и посмотрите, что пишется в логах в этот момент.

Если приложение падает в момент запроса, в логах отобразится ошибка (например, `panic` в Go или `[error]` в Nginx). 

Какой именно сервис (фронтенд или бэкенд) выдал эту ошибку, и что сейчас написано в его **Deploy Logs**?