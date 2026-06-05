# Frontend-ті Vercel-ге, ал Backend-ті Railway-ге орналастыру нұсқаулығы

Бұл нұсқаулық жобаның жаңа архитектурасын баптауды сипаттайды:
- **Frontend (Vue 3 + Vite):** Vercel платформасында
- **Backend (Go) және PostgreSQL:** Railway платформасында

---

## 1. Railway-де бэкэндті дайындау / Prepare Backend on Railway

1. Railway-дегі **backend** сервисінің сыртқы сілтемесін (Public URL) көшіріп алыңыз.
   - Мысалы: `https://backend-production-3daa.up.railway.app`
2. **Backend** сервисінің **Variables** қойындысына өтіп, келесі айнымалыны жаңартыңыз:
   - `CORS_ALLOWED_ORIGINS`: `*` немесе сіздің болашақ Vercel сілтемеңіз (мысалы: `https://operation-plan.vercel.app`).
   > [!TIP]
   > Егер `*` мәнін қойсаңыз, кез келген доменнен сұраныстарды қабылдай береді. Бұл CORS қателігін толығымен алдын алады.

---

## 2. Frontend-ті Vercel-ге орналастыру / Deploy Frontend on Vercel

1. [Vercel](https://vercel.com) сайтына кіріп, **Add New** -> **Project** батырмасын басыңыз.
2. Өз GitHub репозиториіңізді таңдап, **Import** басыңыз.
3. **Configure Project** терезесінде келесі баптауларды міндетті түрде енгізіңіз:
   - **Framework Preset:** `Vite` (немесе автоматты түрде таңдалады).
   - **Root Directory:** **`frontend`** деп көрсетіңіз (бұл өте маңызды, себебі Vue жобасы `frontend` папкасының ішінде орналасқан).
4. **Environment Variables** (Орталық айнымалылар) бөлімін ашып, келесі мәнді қосыңыз:
   - **Name:** `VITE_API_BASE_URL`
   - **Value:** `https://ваша-бэкэнд-сілтеме.up.railway.app` (Мысалы: `https://backend-production-3daa.up.railway.app` — соңында қиғаш сызық `/` **болмауы** керек).
5. **Deploy** батырмасын басыңыз.

---

## 3. SPA роутингті шешу / SPA Routing Configuration

Біз сіз үшін `frontend/vercel.json` файлын дайындап қойдық. Бұл файл Vercel-ге кез келген бетті жаңартқанда (мысалы, `/plans` немесе `/tasks` бетінде `F5` басқанда) **404 Page Not Found** қатесін бермей, Vue Router-дің ішкі маршруттарын дұрыс жүктеуге көмектеседі.

Кодты GitHub-қа жіберген (push) кезде ол автоматты түрде Vercel-де қолданылатын болады.
