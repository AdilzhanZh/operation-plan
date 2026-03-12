<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import { createUserRequest, fetchUsersRequest } from '../services/user.service'

const { tr } = useLocale()
const users = ref([])
const loading = ref(false)
const creating = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

function roleLabel(role) {
  if (role === 'admin') return tr('Администратор', 'Әкімші')
  if (role === 'prorector') return tr('Проректор', 'Проректор')
  if (role === 'viewer') return tr('Наблюдатель', 'Бақылаушы')
  return role
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
  return date.toLocaleString(tr('ru-RU', 'kk-KZ'))
}

async function loadUsers() {
  loading.value = true
  clearMessages()

  try {
    const response = await fetchUsersRequest()
    users.value = response.items ?? []
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr('Ошибка загрузки пользователей', 'Пайдаланушыларды жүктеу кезінде қате болды')
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
    successMessage.value = tr('Пользователь успешно создан', 'Пайдаланушы сәтті құрылды')
    resetForm()
    await loadUsers()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr('Ошибка при создании пользователя', 'Пайдаланушыны құру кезінде қате болды')
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
      :title="tr('Пользователи', 'Пайдаланушылар')"
      :subtitle="tr('Управление ролями, создание новых аккаунтов и обзор всей пользовательской базы в одном разделе', 'Рөлдерді басқару, жаңа аккаунт құру және барлық пайдаланушы базасын бір бөлімнен шолу')"
      :eyebrow="tr('Администрирование', 'Әкімшілік')"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="users-top-grid">
      <section class="panel panel-accent users-form-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Новый пользователь', 'Жаңа пайдаланушы') }}</h3>
            <p class="panel-subtitle">{{ tr('Создайте аккаунт и сразу назначьте роль для рабочего процесса.', 'Аккаунт құрып, жұмыс процесіне арналған рөлді бірден тағайындаңыз.') }}</p>
          </div>
          <span class="kicker">{{ tr('Создать', 'Құру') }}</span>
        </div>

        <form class="users-form" @submit.prevent="createUser">
          <label>
            {{ tr('Имя', 'Аты') }}
            <input v-model="form.first_name" type="text" autocomplete="given-name" required />
          </label>
          <label>
            {{ tr('Фамилия', 'Фамилиясы') }}
            <input v-model="form.last_name" type="text" autocomplete="family-name" required />
          </label>
          <label>
            {{ tr('Отчество', 'Тегі') }}
            <input v-model="form.middle_name" type="text" autocomplete="additional-name" />
          </label>
          <label>
            {{ tr('Логин', 'Логин') }}
            <input v-model="form.username" type="text" autocomplete="username" required />
          </label>
          <label>
            {{ tr('Пароль', 'Құпиясөз') }}
            <input v-model="form.password" type="password" autocomplete="new-password" required />
          </label>
          <label>
            {{ tr('Пароль (повтор)', 'Құпиясөз (қайта)') }}
            <input v-model="form.confirm_password" type="password" autocomplete="new-password" required />
          </label>
          <label class="users-role-field">
            {{ tr('Роль', 'Рөлі') }}
            <select v-model="form.role" required>
              <option value="admin">{{ tr('Администратор', 'Әкімші') }}</option>
              <option value="prorector">{{ tr('Проректор', 'Проректор') }}</option>
              <option value="viewer">{{ tr('Наблюдатель', 'Бақылаушы') }}</option>
            </select>
          </label>

          <button type="submit" class="btn btn-primary users-submit" :disabled="creating">
            {{ creating ? tr('Создание...', 'Құрылуда...') : tr('Создать нового пользователя', 'Жаңа пайдаланушы құру') }}
          </button>
        </form>
      </section>

      <section class="panel panel-strong users-stats-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Состав базы', 'База құрамы') }}</h3>
            <p class="panel-subtitle">{{ tr('Моментальный срез по ролям и общему количеству аккаунтов.', 'Рөлдер мен аккаунттардың жалпы саны бойынша жедел шолу.') }}</p>
          </div>
          <span class="kicker">{{ tr('Обзор', 'Шолу') }}</span>
        </div>

        <div class="users-stats-grid">
          <div class="info-card">
            <span>{{ tr('Всего', 'Барлығы') }}</span>
            <strong>{{ stats.total }}</strong>
          </div>
          <div class="info-card">
            <span>{{ tr('Администраторы', 'Әкімшілер') }}</span>
            <strong>{{ stats.admin }}</strong>
          </div>
          <div class="info-card">
            <span>{{ tr('Проректоры', 'Проректорлар') }}</span>
            <strong>{{ stats.prorector }}</strong>
          </div>
          <div class="info-card">
            <span>{{ tr('Наблюдатели', 'Бақылаушылар') }}</span>
            <strong>{{ stats.viewer }}</strong>
          </div>
        </div>
      </section>
    </div>

    <section class="panel panel-strong users-list-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ tr('Пользователи системы', 'Жүйе пайдаланушылары') }}</h3>
          <p class="panel-subtitle">{{ tr('Текущие учетные записи с логинами, ролями и датой создания.', 'Ағымдағы аккаунттар: логин, рөл және құрылған күні.') }}</p>
        </div>
        <span class="kicker">{{ tr('Реестр', 'Тізілім') }}</span>
      </div>

      <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
      <div v-else class="table-wrapper">
        <table class="users-table">
          <thead>
            <tr>
              <th>{{ tr('ID', 'ID') }}</th>
              <th>{{ tr('Полное имя', 'Толық аты') }}</th>
              <th>{{ tr('Логин', 'Логин') }}</th>
              <th>{{ tr('Роль', 'Рөлі') }}</th>
              <th>{{ tr('Пароль', 'Құпиясөз') }}</th>
              <th>{{ tr('Дата создания', 'Құрылған күні') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in users" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.full_name || '—' }}</td>
              <td>{{ item.username }}</td>
              <td>{{ roleLabel(item.role) ?? item.role }}</td>
              <td>{{ item.password_plain || tr('Скрыт', 'Жасырылған') }}</td>
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
