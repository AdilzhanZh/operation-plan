<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { createUserRequest, fetchUsersRequest } from '../services/user.service'

const users = ref([])
const loading = ref(false)
const creating = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

const roleLabels = {
  admin: 'Administrator',
  prorector: 'Prorector',
  viewer: 'Viewer'
}

const form = reactive({
  first_name: '',
  last_name: '',
  middle_name: '',
  username: '',
  password: '',
  confirm_password: '',
  role: 'viewer'
})

const stats = computed(() => ({
  total: users.value.length,
  admin: users.value.filter((item) => item.role === 'admin').length,
  prorector: users.value.filter((item) => item.role === 'prorector').length,
  viewer: users.value.filter((item) => item.role === 'viewer').length
}))

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

function formatDate(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '—'
  }
  return date.toLocaleString('ru-RU')
}

async function loadUsers() {
  loading.value = true
  clearMessages()

  try {
    const response = await fetchUsersRequest()
    users.value = response.items ?? []
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? 'Пайдаланушыларды жүктеу кезінде қате болды'
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.first_name = ''
  form.last_name = ''
  form.middle_name = ''
  form.username = ''
  form.password = ''
  form.confirm_password = ''
  form.role = 'viewer'
}

async function createUser() {
  creating.value = true
  clearMessages()

  try {
    await createUserRequest(form)
    successMessage.value = 'Пайдаланушы сәтті құрылды'
    resetForm()
    await loadUsers()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? 'Пайдаланушы құру кезінде қате болды'
  } finally {
    creating.value = false
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<template>
  <section class="users-page">
    <PageHeader
      title="Users"
      subtitle="Управление ролями, создание новых аккаунтов и обзор всей пользовательской базы в одном разделе"
      eyebrow="Administration"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="users-top-grid">
      <section class="panel panel-accent users-form-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">Новый пользователь</h3>
            <p class="panel-subtitle">Создайте аккаунт и сразу назначьте роль для рабочего процесса.</p>
          </div>
          <span class="kicker">Create</span>
        </div>

        <form class="users-form" @submit.prevent="createUser">
          <label>
            Аты
            <input v-model="form.first_name" type="text" autocomplete="given-name" required />
          </label>
          <label>
            Фамилиясы
            <input v-model="form.last_name" type="text" autocomplete="family-name" required />
          </label>
          <label>
            Тегі
            <input v-model="form.middle_name" type="text" autocomplete="additional-name" />
          </label>
          <label>
            Логин
            <input v-model="form.username" type="text" autocomplete="username" required />
          </label>
          <label>
            Пароль
            <input v-model="form.password" type="password" autocomplete="new-password" required />
          </label>
          <label>
            Пароль (қайта)
            <input v-model="form.confirm_password" type="password" autocomplete="new-password" required />
          </label>
          <label class="users-role-field">
            Рөл
            <select v-model="form.role" required>
              <option value="admin">admin</option>
              <option value="prorector">prorector</option>
              <option value="viewer">viewer</option>
            </select>
          </label>

          <button type="submit" class="btn btn-primary users-submit" :disabled="creating">
            {{ creating ? 'Құрылуда...' : 'Create new user' }}
          </button>
        </form>
      </section>

      <section class="panel panel-strong users-stats-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">Состав базы</h3>
            <p class="panel-subtitle">Моментальный срез по ролям и общему количеству аккаунтов.</p>
          </div>
          <span class="kicker">Overview</span>
        </div>

        <div class="users-stats-grid">
          <div class="info-card">
            <span>Всего</span>
            <strong>{{ stats.total }}</strong>
          </div>
          <div class="info-card">
            <span>Admins</span>
            <strong>{{ stats.admin }}</strong>
          </div>
          <div class="info-card">
            <span>Prorectors</span>
            <strong>{{ stats.prorector }}</strong>
          </div>
          <div class="info-card">
            <span>Viewers</span>
            <strong>{{ stats.viewer }}</strong>
          </div>
        </div>
      </section>
    </div>

    <section class="panel panel-strong users-list-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">Пользователи системы</h3>
          <p class="panel-subtitle">Текущие учетные записи с логинами, ролями и датой создания.</p>
        </div>
        <span class="kicker">Registry</span>
      </div>

      <div v-if="loading" class="empty-state">Жүктелуде...</div>
      <div v-else class="table-wrapper">
        <table class="users-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Толық аты</th>
              <th>Username</th>
              <th>Role</th>
              <th>Password</th>
              <th>Құрылған күні</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in users" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.full_name || '—' }}</td>
              <td>{{ item.username }}</td>
              <td>{{ roleLabels[item.role] ?? item.role }}</td>
              <td>{{ item.password_plain || 'Скрыт' }}</td>
              <td>{{ formatDate(item.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>

<style scoped>
.users-top-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: minmax(0, 1.3fr) minmax(280px, 0.7fr);
}

.users-form-card,
.users-stats-card,
.users-list-card {
  padding: 1.2rem;
}

.users-form {
  display: grid;
  gap: 0.9rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.users-role-field,
.users-submit {
  grid-column: 1 / -1;
}

.users-submit {
  justify-content: center;
}

.users-stats-grid {
  display: grid;
  gap: 0.8rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

@media (max-width: 1080px) {
  .users-top-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .users-form,
  .users-stats-grid {
    grid-template-columns: 1fr;
  }

  .users-role-field,
  .users-submit {
    grid-column: auto;
  }
}
</style>
