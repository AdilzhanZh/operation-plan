<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import { changePasswordRequest, getMeRequest } from '../services/auth.service'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const authStore = useAuthStore()

const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const showPasswordForm = ref(false)

const roleLabels = {
  admin: 'Administrator',
  prorector: 'Prorector',
  viewer: 'Viewer'
}

const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const profileFields = computed(() => [
  { label: 'Аты', value: authStore.user?.first_name || '—' },
  { label: 'Фамилиясы', value: authStore.user?.last_name || '—' },
  { label: 'Тегі', value: authStore.user?.middle_name || '—' },
  { label: 'Толық аты', value: authStore.user?.full_name || '—' },
  { label: 'Логин', value: authStore.user?.username || '—' },
  { label: 'Рөл', value: roleLabels[authStore.user?.role] ?? authStore.user?.role ?? '—' }
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
    errorMessage.value = error?.response?.data?.error ?? 'Профильді жүктеу кезінде қате болды'
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
    errorMessage.value = error?.response?.data?.error ?? 'Пароль өзгерту кезінде қате болды'
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
      title="Профиль"
      subtitle="Личные данные пользователя и безопасная смена пароля без перехода в отдельные настройки"
      eyebrow="Account"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="profile-grid">
      <section class="panel panel-strong profile-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">Карточка пользователя</h3>
            <p class="panel-subtitle">Базовые учетные данные, роль и рабочая идентификация в системе.</p>
          </div>
          <span class="kicker">Profile</span>
        </div>

        <div v-if="loading" class="empty-state">Жүктелуде...</div>
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
            <h3 class="panel-title">Безопасность доступа</h3>
            <p class="panel-subtitle">Обновление пароля выполняется здесь же, без отдельного маршрута.</p>
          </div>
          <span class="kicker">Security</span>
        </div>

        <div v-if="!showPasswordForm" class="stack-sm">
          <p class="profile-note">
            Меняйте пароль сразу после первого входа и каждый раз, когда доступ получает новый ответственный.
          </p>
          <button type="button" class="btn btn-primary profile-action" @click="openPasswordForm">
            Сменить пароль
          </button>
        </div>

        <form v-else class="profile-password-form" @submit.prevent="changePassword">
          <label>
            Ескі пароль
            <input v-model="form.old_password" type="password" autocomplete="current-password" required />
          </label>
          <label>
            Жаңа пароль
            <input v-model="form.new_password" type="password" autocomplete="new-password" required />
          </label>
          <label>
            Жаңа пароль (қайталау)
            <input v-model="form.confirm_password" type="password" autocomplete="new-password" required />
          </label>

          <div class="profile-actions-row">
            <button type="button" class="btn btn-ghost" @click="closePasswordForm">
              Жабу
            </button>
            <button type="submit" class="btn btn-primary" :disabled="saving">
              {{ saving ? 'Сақталуда...' : 'Парольді ауыстыру' }}
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
