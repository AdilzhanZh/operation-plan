# Oper Plan: ағымдағы техникалық спецификация

Құжат мәртебесі: `current-state spec`

Негізі: репозиторийдің ағымдағы коды мен конфигурациясы  
Күні: `2026-03-27`

## 1. Қысқаша сипаттама

`Oper Plan` жобасы даму бағдарламасы бойынша:

- жылдық мақсатты индикаторларды жүргізуге,
- сол индикаторлар негізінде жұмыс жоспарын қалыптастыруға,
- жауаптыларды бекітуге,
- есептерді қабылдап/қайтаруға,
- орындалу барысын рөлдер бойынша бақылауға

арналған web-жүйе.

Жоба екі негізгі бөліктен тұрады:

- `backend` — `Go + Gin + PostgreSQL`
- `frontend` — `Vue 3 + Vite`

Қазіргі код күйінде жүйенің негізгі бизнес-өзегі:

- `planning period` модулі,
- `plan items` модулі,
- `report submission / review` модулі,
- `user / auth` модулі.

## 2. Жүйенің мақсаты

Жүйенің мақсаты:

- даму бағдарламасының индикаторларын жылдар бойынша сақтау;
- әр жылға жоспар элементтерін толықтыру;
- жауапты қызметкерлерді бекіту;
- қызметкерлерден есеп қабылдау;
- әкімші деңгейінде есепті келісу немесе қайтару;
- статус, мерзім және прогресс бойынша көрнекі бақылау беру.

## 3. Ағымдағы архитектура

### 3.1 Жоғары деңгей

Жүйе классикалық `SPA + REST API + PostgreSQL` архитектурасында құрылған.

Компоненттер:

1. `Frontend`
   - Vue 3 SPA
   - Axios арқылы backend API-ге қосылады
   - авторизация token-і `localStorage` ішінде сақталады

2. `Backend`
   - Gin HTTP server
   - Bearer token арқылы сессия тексереді
   - SQL және ішінара GORM migration қолданады
   - файлдарды локал storage-қа сақтайды

3. `Database`
   - PostgreSQL
   - негізгі бизнес кестелері мен auth кестелері backend startup кезінде құрылады/жаңартылады

4. `File storage`
   - жүктелген файлдар `backend/storage` ішінде сақталады

### 3.2 Docker архитектурасы

`docker-compose.yml` бойынша 3 сервис қолданылады:

- `postgres`
- `backend`
- `frontend`

Байланыс:

- `frontend` -> `/api` -> `backend:8080`
- `backend` -> `postgres:5432`
- `backend` -> локал volume `./backend/storage:/app/storage`

## 4. Технологиялық стек

### 4.1 Backend

- `Go 1.25`
- `Gin`
- `GORM`
- `PostgreSQL`
- `swaggo` / Swagger
- `excelize` — Excel импортын оқу үшін
- `bcrypt` — пароль хэштеу үшін

### 4.2 Frontend

- `Vue 3`
- `Vue Router`
- `Pinia`
- `Axios`
- `Vite`

### 4.3 Infra

- `Docker Compose`
- `Nginx` — production frontend serve + `/api` proxy

## 5. Рөлдер және рұқсаттар

Жүйеде 3 негізгі рөл бар:

- `admin`
- `employee`
- `viewer`

### 5.1 Role matrix

| Мүмкіндік | admin | employee | viewer |
| --- | --- | --- | --- |
| Жүйеге кіру | Иә | Иә | Иә |
| Профильді көру | Иә | Иә | Иә |
| Құпиясөз ауыстыру | Иә | Иә | Иә |
| Қолданушылар тізімін көру | Иә | Жоқ | Жоқ |
| Қолданушы құру/өшіру | Иә | Жоқ | Жоқ |
| Тіркелу сұраныстарын approve/reject | Иә | Жоқ | Жоқ |
| Жоспарлы кезеңді көру | Иә | Иә | Жоқ |
| Жоспарлы кезеңге индикатор қосу/өзгерту | Иә | Жоқ | Жоқ |
| Excel импорт жасау | Иә | Жоқ | Жоқ |
| Жоспар индикаторларын көру | Иә | Иә | Иә |
| Жоспар индикаторларын редакциялау | Иә | Жоқ | Жоқ |
| Жауаптыларды бекіту | Иә | Жоқ | Жоқ |
| Есеп жіберу | Жоқ | Иә | Жоқ |
| Есеп тарихын көру | Иә | Иә | Шектеулі/жоқ |
| Есепті approve/reject | Иә | Жоқ | Жоқ |
| Қайтарылған есепті қайта жіберу | Жоқ | Иә | Жоқ |
| Dashboard көру | Иә | Иә | Иә |
| Program execution беті | Иә | Иә | Жоқ |

Ескерту:

- frontend маршруттары рөлге қарай шектеледі;
- backend те қосымша түрде `AuthRequired` және `RequireRoles(...)` арқылы қолжетімділікті қорғайды.

## 6. Негізгі функционалдық модульдер

### 6.1 Аутентификация және сессия

Қолдайтын сценарийлер:

- `login`
- `logout`
- `me`
- `change password`
- email арқылы тіркелу коды
- email арқылы парольді қалпына келтіру

Техникалық ерекшеліктер:

- авторизация `Bearer token` арқылы жүреді;
- token `user_sessions` кестесінде сақталады;
- сессияның TTL-ы env арқылы реттеледі;
- әр request кезінде token база арқылы тексеріледі;
- bootstrap admin механизмі бар.

### 6.2 Тіркелу модулі

Қазіргі логика:

1. Пайдаланушы тіркелу формасын толтырады.
2. Backend email-ға `6 таңбалы код` жібереді.
3. Код расталғаннан кейін бірден user жасалмайды.
4. Алдымен `registration_requests` ішінде сұраныс жасалады.
5. `admin` бұл сұранысты approve немесе reject етеді.
6. Approve кезінде нақты user құрылады.

### 6.3 Парольді қалпына келтіру

Қазіргі логика:

1. Пайдаланушы email енгізеді.
2. Email-ға код жіберіледі.
3. Код расталған соң `reset_token` беріледі.
4. Сол token арқылы жаңа пароль орнатылады.

### 6.4 Қолданушыларды басқару

Admin үшін мүмкіндіктер:

- user list
- role/filter/search
- user create
- user delete
- registration request list
- approve/reject request
- employee list-ін жеке endpoint арқылы алу

### 6.5 Жоспарлы кезең модулі

Модуль мақсаты:

- мақсатты индикаторларды сақтау;
- әр индикаторға бірнеше жылдық мән бекіту;
- бағыт бойынша бөлу;
- Excel файлынан импорт жасау.

Қолдайтын өрістер:

- `target_indicator`
- `unit`
- `direction`
- `year_values`

`direction` үшін рұқсат етілген 3 мән бар:

- `Академическое превосходство и интернационализация образования`
- `РАЗВИТИЕ НАУКИ И МЕЖДУНАРОДНОГО СОТРУДНИЧЕСТВА`
- `ЦИФРОВИЗАЦИЯ И МОДЕРНИЗАЦИЯ ИНФРАСТРУКТУРЫ`

### 6.6 Жоспарлар және индикаторлар модулі

Бұл модуль жоспарлы кезеңдегі индикаторлардан белгілі бір жыл үшін жұмыс жоспарларын құрады.

Admin толтыра алатын өрістер:

- `development_indicator`
- `evaluation_formula`
- `activities`
- `execution_start_date`
- `execution_end_date`
- `responsible_user_ids`

Жоспар статустары:

- `not_filled`
- `upcoming`
- `in_progress`
- `overdue`

Есеп статустары:

- `pending`
- `completed`
- `rejected`

### 6.7 Есептерді жіберу және келісу

`employee` тарапынан:

- өзіне бекітілген индикаторлар бойынша есеп жібере алады;
- мәтіндік есеп тіркей алады;
- бір немесе бірнеше файл тіркей алады;
- reject болған есепті қайта жібере алады;
- өз есептерінің тарихын қарай алады.

`admin` тарапынан:

- жіберілген есептерді тізімдей алады;
- есеп мәтінін қарай алады;
- файлдарды жүктей алады;
- есепті `approve` немесе `reject` ете алады;
- approve кезінде қорытынды формула жаза алады;
- reject кезінде ескерту/себеп жаза алады.

### 6.8 Dashboard

Dashboard модулі жыл бойынша агрегатталған көрініс береді.

Көрсетілетін агрегаттар:

- барлық индикаторлар саны
- қабылданғандар
- тексерістегілер
- толтырылмағандар
- жұмыстағылар
- мерзімі өткендер

`employee` үшін dashboard тек өзіне қатысты деректерге сүйенеді.

### 6.9 Program Execution

Бұл бет есептерді қарауға арналған бөлек жұмыс аймағы.

Admin режимі:

- `pending`
- `completed`
- `rejected`

employee режимі:

- `pending`
- `completed`
- `rejected`

Бұл модульде:

- report details modal,
- review modal,
- rejected report resubmission flow

бар.

### 6.10 Профиль және интерфейс

Қолдайтын мүмкіндіктер:

- өз профилін көру
- парольді ауыстыру
- RU/KZ тілін ауыстыру

Тіл сақтау механизмі:

- `localStorage` (`oper-plan-locale`)

## 7. Frontend құрылымы

Негізгі маршруттар:

- `/login`
- `/register`
- `/forgot-password`
- `/dashboard`
- `/profile`
- `/users`
- `/plans`
- `/planning-period`
- `/program-execution`

Негізгі беттер:

- `LoginPage.vue`
- `RegisterPage.vue`
- `ForgotPasswordPage.vue`
- `DashboardPage.vue`
- `ProfilePage.vue`
- `UsersPage.vue`
- `PlansPage.vue`
- `PlanningPeriodPage.vue`
- `ProgramExecutionPage.vue`
- `NotFoundPage.vue`

Негізгі frontend модульдері:

- `router/index.js` — route guard және role-based navigation
- `store/auth.js` — token/user сақтау
- `services/*.js` — API клиенттері
- `composables/useLocale.js` — локализация
- `App.vue` — topbar shell, profile menu, language switch

## 8. Backend модульдері

Негізгі backend пакеттері:

- `auth`
- `user`
- `plan`
- `period`
- `task`
- `report`
- `middleware`
- `config`
- `database`
- `handler`
- `server`

### 8.1 Auth API

- `POST /login`
- `POST /logout`
- `GET /me`
- `POST /change-password`
- `POST /register`
- `POST /register/request-code`
- `POST /register/verify-code`
- `POST /password-reset/request-code`
- `POST /password-reset/verify-code`
- `POST /password-reset/confirm`

### 8.2 User API

- `GET /users`
- `GET /users/employees`
- `POST /users`
- `DELETE /users/:id`
- `GET /registration-requests`
- `PATCH /registration-requests/:id/approve`
- `PATCH /registration-requests/:id/reject`

### 8.3 Planning Period API

- `GET /planning-period`
- `POST /planning-period`
- `PATCH /planning-period/:id`
- `POST /planning-period/import`

### 8.4 Plans API

- `GET /plans`
- `POST /plans`
- `GET /plan-records/:id`
- `PATCH /plan-records/:id/status`
- `GET /plans/years`
- `GET /plans/indicators`
- `PUT /plans/indicators/:indicator_id`
- `POST /plans/indicators/:indicator_id/report`
- `GET /plans/reports`
- `PATCH /plans/reports/:report_id`
- `PATCH /plans/reports/:report_id/review`
- `GET /plans/reports/files/:file_id/download`

### 8.5 Legacy Tasks/Reports API

Код базасында қосымша legacy task/report модулі де бар:

- `GET /tasks`
- `GET /tasks/:id`
- `POST /tasks`
- `PATCH /tasks/:id`
- `PATCH /tasks/:id/status`
- `DELETE /tasks/:id`
- `POST /tasks/:id/report`
- `GET /tasks/:id/reports`

Ескерту:

- бұл модуль backend ішінде бар;
- frontend-тің ағымдағы негізгі navigation-ы бұл API-ге сүйенбейді;
- жобаның қазіргі негізгі бизнес-ағыны `planning-period -> plan-items -> reports` моделіне байланған.

## 9. Негізгі бизнес-процестер

### 9.1 Login flow

1. User `/login` формасын толтырады.
2. Backend username/password тексереді.
3. Пароль `bcrypt` арқылы тексеріледі.
4. Token генерацияланып, `user_sessions` ішіне жазылады.
5. Frontend token мен user-ді store/localStorage-қа сақтайды.

### 9.2 Registration flow

1. User тіркелу формасын толтырады.
2. Backend деректерді валидтейді.
3. Email-ға код жібереді.
4. User кодты енгізеді.
5. Backend verification кодты consume етеді.
6. `registration_requests` ішінде pending request құрады.
7. Admin request-ті approve/reject етеді.
8. Approve болса нақты `users` жазбасы жасалады.

### 9.3 Password reset flow

1. User email жібереді.
2. Backend код жібереді.
3. User кодты растайды.
4. Backend reset session/token береді.
5. User жаңа пароль қояды.
6. Пароль hash түрінде сақталады.

### 9.4 Planning period import flow

1. Admin Excel файл жүктейді.
2. Backend workbook-ты `excelize` арқылы оқиды.
3. Индикатор, өлшем бірлік және жылдық мәндер парсингтен өтеді.
4. Жаңа жолдар құрылады немесе барлары жаңартылады.

### 9.5 Plan item preparation flow

1. Admin қажетті жылды таңдайды.
2. Жүйе сол жылға тиесілі индикаторларды шығарады.
3. Admin индикатор үшін жұмыс мәліметтерін толтырады.
4. Жауапты `employee`-лар бекітіледі.
5. Жоспар статусы автоматты түрде есептеледі.

### 9.6 Report submission flow

1. employee өзіне бекітілген индикаторды ашады.
2. Есеп мәтінін енгізеді.
3. Бір немесе бірнеше файл тіркейді.
4. Backend файлдарды валидтейді.
5. Report submission және files жазбалары жасалады.
6. Есеп статусы `pending` болады.

### 9.7 Report review flow

1. Admin pending есепті ашады.
2. Есеп мәтінін және attached файлдарды қарайды.
3. Екі шешімнің бірін қабылдайды:
   - `approve`
   - `reject`
4. Approve болса қорытынды формула жазылады, статус `completed`.
5. Reject болса себеп жазылады, статус `rejected`.
6. Өзгеріс тарихы `report_status_history` ішінде сақталады.

## 10. Дерекқор моделі

### 10.1 Auth және user кестелері

- `users`
- `user_sessions`
- `registration_requests`
- `email_verification_codes`
- `password_reset_sessions`

### 10.2 Жоспарлау ядросының кестелері

Қазіргі негізгі ядро:

- `planning_period_indicators`
- `indicator_year_targets`
- `plan_items`
- `plan_item_responsibles`
- `report_submissions`
- `report_files`
- `report_status_history`

Мақсаты:

- `planning_period_indicators` — индикатор карточкасы
- `indicator_year_targets` — жыл бойынша target мәндер
- `plan_items` — индикатордың нақты жылдық жұмыс элементі
- `plan_item_responsibles` — жауаптылар байланысы
- `report_submissions` — жіберілген есептер
- `report_files` — есеп файлдары
- `report_status_history` — статус тарихы

### 10.3 Legacy/transition кестелері

Код базасында келесі кестелер де бар:

- `plans`
- `tasks`
- `reports`
- `task_logs`
- `plan_indicator_details`
- `plan_indicator_reports`
- `plan_indicator_report_files`

Бұл кестелердің бір бөлігі бұрынғы модельді немесе өтпелі миграция логикасын көрсетеді. Ағымдағы негізгі frontend ағымы normalized кестелерге көбірек сүйенеді.

## 11. Файлдармен жұмыс

Қолдайтын кеңейтімдер:

- `.pdf`
- `.doc`
- `.docx`
- `.xls`
- `.xlsx`
- `.ppt`
- `.pptx`
- `.jpg`
- `.jpeg`
- `.png`

Шектеу:

- максимум `20 MB`

Сақтау орны:

- backend локал storage
- `backend/storage`

Маңызды шектеу:

- бұл cloud persistent object storage емес;
- контейнер/сервер ауысса, файлдардың сақталуы deployment тәсіліне тәуелді.

## 12. Локализация

Қолдайтын тілдер:

- `ru`
- `kz`

Ерекшеліктер:

- browser locale анықтайды;
- user таңдауы `localStorage` ішінде сақталады;
- барлық негізгі UI мәтіндері `tr(ru, kz)` үлгісімен берілген.

## 13. Конфигурация және environment variables

Негізгі env параметрлер:

- `PORT`
- `LOG_LEVEL`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`
- `DB_TIMEZONE`
- `CORS_ALLOWED_ORIGINS`
- `SESSION_TTL_HOURS`
- `BOOTSTRAP_ADMIN_USERNAME`
- `BOOTSTRAP_ADMIN_PASSWORD`
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `SMTP_FROM_EMAIL`
- `SMTP_FROM_NAME`
- `OTP_TTL_MINUTES`
- `RESET_SESSION_TTL_MINUTES`

Frontend env:

- `VITE_API_BASE_URL`
- `VITE_API_PROXY_TARGET`

## 14. Іске қосу және deploy

### 14.1 Local/Docker run

Ұсынылатын негізгі тәсіл:

```bash
docker compose up --build -d
```

Қолжетімді нүктелер:

- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`
- Health: `http://localhost:8080/healthz`

### 14.2 Backend startup behavior

Backend startup кезінде:

1. env оқылады;
2. PostgreSQL-ге connection ашылады;
3. `AutoMigrate` жүреді;
4. explicit SQL migrations жүреді;
5. route-тар тіркеледі;
6. HTTP server іске қосылады.

## 15. Валидация және қауіпсіздік

Қазіргі кодта бар қорғаныс шаралары:

- password hash (`bcrypt`)
- role-based access control
- bearer token session validation
- email format validation
- password complexity validation
- upload extension validation
- upload size validation
- session TTL

Шектеулер:

- token JWT емес, DB-backed session token;
- rate limit жоқ;
- captcha жоқ;
- audit logging толық емес;
- file antivirus scan жоқ.

## 16. Тесттер және сапа күйі

Ағымдағы репода автоматты тест аз.

Бар тест:

- `backend/internal/period/import_parser_test.go`

Бұл тест негізінен:

- Excel import parser-дің string target мәндерін қабылдауын
- `-` және `1200-1400` сияқты мәндерді бұзбай оқуын

тексереді.

Қорытынды:

- бизнес-ағындар кодпен іске асқан;
- бірақ automated coverage төмен.

## 17. Ағымдағы техникалық шектеулер

1. Email-ға тәуелді сценарийлер SMTP конфигурациясыз жұмыс істемейді.
2. Файл сақтау локал filesystem-ге байланған.
3. Жоба ішінде legacy және current data model қатар өмір сүріп тұр.
4. Task API мен жаңа plan/report API қатар бар, бірақ негізгі UI жаңа ағынға бағытталған.
5. Viewer рөлі негізінен read-only режимде.
6. Deploy үшін тұрақты cloud storage және managed Postgres қажет болуы мүмкін.
7. Full e2e және integration test жоқ.

## 18. Ағымдағы күй бойынша қысқаша қорытынды

Жобаның қазіргі күйі demo және ішкі пайдалануға жеткілікті деңгейде жұмыс істейтін ақпараттық-жоспарлау жүйесін сипаттайды.

Негізгі дайын бөліктер:

- login/logout/auth
- registration approval flow
- password reset flow
- user management
- planning period management
- yearly plan preparation
- report submission/review
- dashboard
- bilingual UI
- dockerized local deployment

Ағымдағы өзек:

`planning_period_indicators -> indicator_year_targets -> plan_items -> report_submissions`

Яғни жоба қазір “жылдық индикатор -> жұмыс жоспары -> есеп -> тексеру” тізбегін жабады.
