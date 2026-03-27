<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import { changePasswordRequest, getMeRequest } from '../services/auth.service'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const authStore = useAuthStore()
const { tr } = useLocale()

const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const showPasswordForm = ref(false)

function roleLabel(role) {
  if (role === 'admin') return tr('Администратор', 'Әкімші')
  if (role === 'prorector') return tr('Сотрудник', 'Қызметкер')
  if (role === 'viewer') return tr('Наблюдатель', 'Бақылаушы')
  return role
}

const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const profileFields = computed(() => [
  { label: tr('Имя', 'Аты'), value: authStore.user?.first_name || '—' },
  { label: tr('Фамилия', 'Фамилиясы'), value: authStore.user?.last_name || '—' },
  { label: tr('Отчество', 'Тегі'), value: authStore.user?.middle_name || '—' },
  { label: tr('Полное имя', 'Толық аты'), value: authStore.user?.full_name || '—' },
  { label: tr('Логин', 'Логин'), value: authStore.user?.username || '—' },
  { label: 'Email', value: authStore.user?.email || '—' },
  { label: tr('Должность', 'Лауазымы'), value: authStore.user?.position || '—' },
  { label: tr('Роль', 'Рөлі'), value: roleLabel(authStore.user?.role) || '—' }
])

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

async function loadProfile() {
  loading.value = true
  clearMessages()

  try {
    const response = await getMeRequest()
    authStore.setUser(response.user)
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr('Ошибка загрузки профиля', 'Профильді жүктеу кезінде қате болды')
  } finally {
    loading.value = false
  }
}

async function changePassword() {
  saving.value = true
  clearMessages()

  try {
    await changePasswordRequest(form)
    form.old_password = ''
    form.new_password = ''
    form.confirm_password = ''
    showPasswordForm.value = false
    authStore.logout()
    await router.push({ name: 'login', query: { reauth: '1' } })
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr('Ошибка при смене пароля', 'Құпиясөзді өзгерту кезінде қате болды')
  } finally {
    saving.value = false
  }
}

function openPasswordForm() {
  clearMessages()
  showPasswordForm.value = true
}

function closePasswordForm() {
  form.old_password = ''
  form.new_password = ''
  form.confirm_password = ''
  showPasswordForm.value = false
}

onMounted(() => {
  loadProfile()
})
</script>

<template>
  <section class="profile-page">
    <PageHeader
      :title="tr('Профиль', 'Профиль')"
      :subtitle="tr('Личные данные пользователя и безопасная смена пароля без перехода в отдельные настройки', 'Пайдаланушы деректері және бөлек бетке өтпей қауіпсіз құпиясөз ауыстыру')"
      :eyebrow="tr('Аккаунт', 'Аккаунт')"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="profile-grid">
      <section class="panel panel-strong profile-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Карточка пользователя', 'Пайдаланушы картасы') }}</h3>
            <p class="panel-subtitle">{{ tr('Базовые учетные данные, роль и рабочая идентификация в системе.', 'Жүйедегі негізгі деректер, рөл және жұмыс идентификациясы.') }}</p>
          </div>
          <span class="kicker">{{ tr('Профиль', 'Профиль') }}</span>
        </div>

        <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
        <div v-else class="info-grid">
          <div v-for="item in profileFields" :key="item.label" class="info-card">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </section>

      <section class="panel panel-accent profile-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Безопасность доступа', 'Қолжетімділік қауіпсіздігі') }}</h3>
            <p class="panel-subtitle">{{ tr('Обновление пароля выполняется здесь же, без отдельного маршрута.', 'Құпиясөз осы жерде, бөлек маршрутсыз жаңартылады.') }}</p>
          </div>
          <span class="kicker">{{ tr('Безопасность', 'Қауіпсіздік') }}</span>
        </div>

        <div v-if="!showPasswordForm" class="stack-sm">
          <p class="profile-note">
            {{ tr('Меняйте пароль сразу после первого входа и каждый раз, когда доступ получает новый ответственный.', 'Алғашқы кіруден кейін және жаңа жауаптыға қолжетімділік берілген сайын құпиясөзді жаңартыңыз.') }}
          </p>
          <button type="button" class="btn btn-primary profile-action" @click="openPasswordForm">
            {{ tr('Сменить пароль', 'Құпиясөзді ауыстыру') }}
          </button>
        </div>

        <form v-else class="profile-password-form" @submit.prevent="changePassword">
          <label>
            {{ tr('Старый пароль', 'Ескі құпиясөз') }}
            <input v-model="form.old_password" type="password" autocomplete="current-password" required />
          </label>
          <label>
            {{ tr('Новый пароль', 'Жаңа құпиясөз') }}
            <input v-model="form.new_password" type="password" autocomplete="new-password" required />
          </label>
          <label>
            {{ tr('Новый пароль (повтор)', 'Жаңа құпиясөз (қайталау)') }}
            <input v-model="form.confirm_password" type="password" autocomplete="new-password" required />
          </label>

          <div class="profile-actions-row">
            <button type="button" class="btn btn-ghost" @click="closePasswordForm">
              {{ tr('Закрыть', 'Жабу') }}
            </button>
            <button type="submit" class="btn btn-primary" :disabled="saving">
              {{ saving ? tr('Сохранение...', 'Сақталуда...') : tr('Обновить пароль', 'Құпиясөзді жаңарту') }}
            </button>
          </div>
        </form>
      </section>
    </div>
  </section>
</template>

<style scoped>
.profile-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
}

.profile-card {
  padding: 1.2rem;
}

.profile-note {
  margin: 0;
  color: var(--muted);
}

.profile-action {
  justify-content: center;
}

.profile-password-form {
  display: grid;
  gap: 0.9rem;
}

.profile-actions-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

@media (max-width: 960px) {
  .profile-grid {
    grid-template-columns: 1fr;
  }
}
</style>
