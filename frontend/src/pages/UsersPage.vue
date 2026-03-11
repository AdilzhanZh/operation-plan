<script setup>
import { onMounted, reactive, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { createUserRequest, fetchUsersRequest } from '../services/user.service'

const users = ref([])
const loading = ref(false)
const creating = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

const form = reactive({
  first_name: '',
  last_name: '',
  middle_name: '',
  username: '',
  password: '',
  confirm_password: '',
  role: 'viewer'
})

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
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
    <PageHeader title="Users" subtitle="Жүйедегі барлық пайдаланушылар тізімі" />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="card">
      <h3>Create new user</h3>

      <form class="create-form" @submit.prevent="createUser">
        <label>
          Аты
          <input v-model="form.first_name" type="text" required />
        </label>
        <label>
          Фамилиясы
          <input v-model="form.last_name" type="text" required />
        </label>
        <label>
          Тегі
          <input v-model="form.middle_name" type="text" />
        </label>
        <label>
          Логин (username)
          <input v-model="form.username" type="text" required />
        </label>
        <label>
          Пароль
          <input v-model="form.password" type="password" required />
        </label>
        <label>
          Жаңа пароль (қайта)
          <input v-model="form.confirm_password" type="password" required />
        </label>
        <label>
          Рөл
          <select v-model="form.role" required>
            <option value="admin">admin</option>
            <option value="prorector">prorector</option>
            <option value="viewer">viewer</option>
          </select>
        </label>

        <button type="submit" :disabled="creating">
          {{ creating ? 'Құрылуда...' : 'Create new user' }}
        </button>
      </form>
    </div>

    <div class="card">
      <h3>Пайдаланушылар тізімі</h3>

      <div v-if="loading">Жүктелуде...</div>
      <div v-else class="table-wrapper">
        <table class="users-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Толық аты</th>
              <th>Username</th>
              <th>Role</th>
              <th>Password (hidden)</th>
              <th>Құрылған күні</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in users" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.full_name }}</td>
              <td>{{ item.username }}</td>
              <td>{{ item.role }}</td>
              <td>{{ item.password_plain }}</td>
              <td>{{ new Date(item.created_at).toLocaleString() }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<style scoped>
.users-page {
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

.create-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.65rem;
}

label {
  display: grid;
  gap: 0.35rem;
  font-size: 0.92rem;
}

input,
select,
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

.table-wrapper {
  overflow-x: auto;
}

.users-table {
  width: 100%;
  border-collapse: collapse;
  min-width: 780px;
}

.users-table th,
.users-table td {
  border: 1px solid #e2e8f0;
  padding: 0.55rem 0.7rem;
  text-align: left;
}

.users-table thead th {
  background: #f8fafc;
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
