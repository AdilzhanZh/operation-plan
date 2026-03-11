<script setup>
import { onMounted, reactive, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { changePasswordRequest, getMeRequest } from '../services/auth.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()

const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const showPasswordForm = ref(false)

const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

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
    successMessage.value = 'Пароль сәтті өзгертілді'
    form.old_password = ''
    form.new_password = ''
    form.confirm_password = ''
    showPasswordForm.value = false
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
    <PageHeader title="Профиль" subtitle="Пайдаланушы деректері және пароль ауыстыру" />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="card">
      <h3>Пайдаланушы деректері</h3>
      <div v-if="loading">Жүктелуде...</div>
      <div v-else class="grid">
        <div><b>Аты:</b> {{ authStore.user?.first_name || '—' }}</div>
        <div><b>Фамилиясы:</b> {{ authStore.user?.last_name || '—' }}</div>
        <div><b>Тегі:</b> {{ authStore.user?.middle_name || '—' }}</div>
        <div><b>Толық аты:</b> {{ authStore.user?.full_name || '—' }}</div>
        <div><b>Логин:</b> {{ authStore.user?.username || '—' }}</div>
        <div><b>Рөлі:</b> {{ authStore.user?.role || '—' }}</div>
      </div>
    </div>

    <div class="card">
      <h3>Сменить пароль</h3>
      <button
        v-if="!showPasswordForm"
        type="button"
        class="toggle-password-btn"
        @click="openPasswordForm"
      >
        Сменить пароль
      </button>

      <form v-else class="password-form" @submit.prevent="changePassword">
        <label>
          Ескі пароль
          <input v-model="form.old_password" type="password" required />
        </label>
        <label>
          Жаңа пароль
          <input v-model="form.new_password" type="password" required />
        </label>
        <label>
          Жаңа пароль (қайталау)
          <input v-model="form.confirm_password" type="password" required />
        </label>

        <div class="form-actions">
          <button type="button" class="cancel-btn" @click="closePasswordForm">
            Жабу
          </button>
          <button type="submit" :disabled="saving">
            {{ saving ? 'Сақталуда...' : 'Парольді ауыстыру' }}
          </button>
        </div>
      </form>
    </div>
  </section>
</template>

<style scoped>
.profile-page {
  display: grid;
  gap: 0.9rem;
}

.card {
  border: 1px solid #dbe2ea;
  border-radius: 10px;
  padding: 1rem;
  background: #ffffff;
}

.card h3 {
  margin: 0 0 0.8rem;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.65rem;
}

.password-form {
  display: grid;
  gap: 0.7rem;
  max-width: 520px;
}

.toggle-password-btn {
  max-width: 220px;
}

label {
  display: grid;
  gap: 0.35rem;
  font-size: 0.92rem;
}

.form-actions {
  display: flex;
  align-items: center;
  gap: 0.55rem;
}

.cancel-btn {
  border: 1px solid #cbd5e1;
  background: #ffffff;
  color: #0f172a;
}

input,
button {
  border-radius: 8px;
  border: 1px solid #cbd5e1;
  padding: 0.55rem 0.75rem;
  font: inherit;
}

button {
  border: none;
  background: #0f172a;
  color: #f8fafc;
  cursor: pointer;
  font-weight: 600;
}

button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.message {
  margin: 0;
  padding: 0.65rem 0.85rem;
  border-radius: 8px;
}

.message-error {
  background: #fee2e2;
  color: #991b1b;
}

.message-success {
  background: #dcfce7;
  color: #166534;
}
</style>
