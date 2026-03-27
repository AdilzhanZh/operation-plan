<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useAuthStore } from '../store/auth'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import {
  approveRegistrationRequest,
  createUserRequest,
  deleteUserRequest,
  fetchRegistrationRequestsRequest,
  fetchUsersRequest,
  rejectRegistrationRequest
} from '../services/user.service'

const { tr } = useLocale()
const authStore = useAuthStore()

const activeTab = ref('users')
const users = ref([])
const registrationRequests = ref([])
const loadingUsers = ref(false)
const loadingRequests = ref(false)
const creating = ref(false)
const requestActionLoading = ref(false)
const deleteLoading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const requestModalOpen = ref(false)
const deleteModalOpen = ref(false)
const requestAction = ref('approve')
const selectedRequest = ref(null)
const selectedUser = ref(null)
const rejectReason = ref('')

const form = reactive({
  first_name: '',
  last_name: '',
  middle_name: '',
  position: '',
  email: '',
  username: '',
  password: '',
  confirm_password: '',
  role: 'viewer'
})

function roleLabel(role) {
  if (role === 'admin') return tr('Администратор', 'Әкімші')
  if (role === 'prorector') return tr('Сотрудник', 'Қызметкер')
  if (role === 'viewer') return tr('Наблюдатель', 'Бақылаушы')
  return role
}

function requestStatusLabel(status) {
  if (status === 'pending') return tr('Ожидает решения', 'Шешім күтілуде')
  if (status === 'approved') return tr('Одобрено', 'Қабылданды')
  if (status === 'rejected') return tr('Отклонено', 'Қабылданбады')
  return status
}

const userStats = computed(() => ({
  total: users.value.length,
  admin: users.value.filter((item) => item.role === 'admin').length,
  prorector: users.value.filter((item) => item.role === 'prorector').length,
  viewer: users.value.filter((item) => item.role === 'viewer').length
}))

const requestStats = computed(() => ({
  total: registrationRequests.value.length,
  pending: registrationRequests.value.filter((item) => item.status === 'pending').length,
  approved: registrationRequests.value.filter((item) => item.status === 'approved').length,
  rejected: registrationRequests.value.filter((item) => item.status === 'rejected').length
}))

const requestDetails = computed(() => {
  if (!selectedRequest.value) {
    return []
  }

  return [
    { label: tr('Полное имя', 'Толық аты'), value: selectedRequest.value.full_name || '—' },
    { label: tr('Имя', 'Аты'), value: selectedRequest.value.first_name || '—' },
    { label: tr('Фамилия', 'Фамилиясы'), value: selectedRequest.value.last_name || '—' },
    { label: tr('Отчество', 'Тегі'), value: selectedRequest.value.middle_name || '—' },
    { label: tr('Логин', 'Логин'), value: selectedRequest.value.username || '—' },
    { label: 'Email', value: selectedRequest.value.email || '—' },
    { label: tr('Должность', 'Лауазымы'), value: selectedRequest.value.position || '—' },
    { label: tr('Роль', 'Рөлі'), value: roleLabel(selectedRequest.value.role) || '—' },
    { label: tr('Статус запроса', 'Сұраныс мәртебесі'), value: requestStatusLabel(selectedRequest.value.status) },
    { label: tr('Создан', 'Құрылған уақыты'), value: formatDate(selectedRequest.value.created_at) }
  ]
})

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
  loadingUsers.value = true

  try {
    const response = await fetchUsersRequest()
    users.value = response.items ?? []
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка загрузки пользователей',
      'Пайдаланушыларды жүктеу кезінде қате болды'
    )
  } finally {
    loadingUsers.value = false
  }
}

async function loadRegistrationRequests() {
  loadingRequests.value = true

  try {
    const response = await fetchRegistrationRequestsRequest()
    registrationRequests.value = response.items ?? []
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка загрузки запросов',
      'Сұраныстарды жүктеу кезінде қате болды'
    )
  } finally {
    loadingRequests.value = false
  }
}

function resetForm() {
  form.first_name = ''
  form.last_name = ''
  form.middle_name = ''
  form.position = ''
  form.email = ''
  form.username = ''
  form.password = ''
  form.confirm_password = ''
  form.role = 'viewer'
}

function canDeleteUser(item) {
  return Number(item?.id) !== Number(authStore.user?.id)
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
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка при создании пользователя',
      'Пайдаланушыны құру кезінде қате болды'
    )
  } finally {
    creating.value = false
  }
}

function openRequestDecision(item, action) {
  clearMessages()
  selectedRequest.value = item
  requestAction.value = action
  rejectReason.value = item.rejection_reason || ''
  requestModalOpen.value = true
}

function closeRequestModal() {
  requestModalOpen.value = false
  selectedRequest.value = null
  rejectReason.value = ''
}

function openDeleteModal(item) {
  clearMessages()
  selectedUser.value = item
  deleteModalOpen.value = true
}

function closeDeleteModal() {
  deleteModalOpen.value = false
  selectedUser.value = null
}

async function submitRequestDecision() {
  if (!selectedRequest.value) {
    return
  }

  requestActionLoading.value = true
  clearMessages()

  try {
    if (requestAction.value === 'approve') {
      await approveRegistrationRequest(selectedRequest.value.id)
      successMessage.value = tr('Запрос успешно одобрен', 'Сұраныс сәтті қабылданды')
    } else {
      await rejectRegistrationRequest(selectedRequest.value.id, {
        reason: rejectReason.value
      })
      successMessage.value = tr('Запрос отклонен', 'Сұраныс қабылданбады')
    }

    closeRequestModal()
    await Promise.all([loadUsers(), loadRegistrationRequests()])
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка при обработке запроса',
      'Сұранысты өңдеу кезінде қате болды'
    )
  } finally {
    requestActionLoading.value = false
  }
}

async function submitDeleteUser() {
  if (!selectedUser.value) {
    return
  }

  deleteLoading.value = true
  clearMessages()

  try {
    await deleteUserRequest(selectedUser.value.id)
    successMessage.value = tr('Пользователь успешно удален', 'Пайдаланушы сәтті өшірілді')
    closeDeleteModal()
    await loadUsers()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка при удалении пользователя',
      'Пайдаланушыны өшіру кезінде қате болды'
    )
  } finally {
    deleteLoading.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadUsers(), loadRegistrationRequests()])
})
</script>

<template>
  <section class="users-page">
    <PageHeader
      :title="tr('Пользователи', 'Пайдаланушылар')"
      :subtitle="tr('Аккаунты системы и все заявки на регистрацию теперь собираются в одном административном контуре.', 'Жүйе аккаунттары мен тіркелу сұраныстары енді бір әкімшілік контурда жиналады.')"
      :eyebrow="tr('Администрирование', 'Әкімшілік')"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="users-tab-strip">
      <button
        type="button"
        class="users-tab-btn"
        :class="{ 'is-active': activeTab === 'users' }"
        @click="activeTab = 'users'"
      >
        {{ tr('Пользователи', 'Пайдаланушылар') }}
      </button>
      <button
        type="button"
        class="users-tab-btn"
        :class="{ 'is-active': activeTab === 'requests' }"
        @click="activeTab = 'requests'"
      >
        {{ tr('Запросы на регистрацию', 'Тіркелу сұраныстары') }}
      </button>
    </div>

    <template v-if="activeTab === 'users'">
      <div class="users-top-grid">
        <section class="panel panel-accent users-form-card">
          <div class="panel-header">
            <div>
              <h3 class="panel-title">{{ tr('Новый пользователь', 'Жаңа пайдаланушы') }}</h3>
              <p class="panel-subtitle">{{ tr('Администратор может создать пользователя напрямую, без ожидания запроса.', 'Әкімші пайдаланушыны сұраныс күтпей тікелей құра алады.') }}</p>
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
              {{ tr('Должность', 'Лауазымы') }}
              <input v-model="form.position" type="text" autocomplete="organization-title" required />
            </label>
            <label class="users-full-row">
              Email
              <input v-model="form.email" type="email" autocomplete="email" required />
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
                <option value="prorector">{{ tr('Сотрудник', 'Қызметкер') }}</option>
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
              <strong>{{ userStats.total }}</strong>
            </div>
            <div class="info-card">
              <span>{{ tr('Администраторы', 'Әкімшілер') }}</span>
              <strong>{{ userStats.admin }}</strong>
            </div>
            <div class="info-card">
              <span>{{ tr('Сотрудники', 'Қызметкерлер') }}</span>
              <strong>{{ userStats.prorector }}</strong>
            </div>
            <div class="info-card">
              <span>{{ tr('Наблюдатели', 'Бақылаушылар') }}</span>
              <strong>{{ userStats.viewer }}</strong>
            </div>
          </div>
        </section>
      </div>

      <section class="panel panel-strong users-list-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Пользователи системы', 'Жүйе пайдаланушылары') }}</h3>
            <p class="panel-subtitle">{{ tr('Актуальные учетные записи: логин, email, должность, роль и дата создания.', 'Ағымдағы аккаунттар: логин, email, лауазым, рөл және құрылған күні.') }}</p>
          </div>
          <span class="kicker">{{ tr('Реестр', 'Тізілім') }}</span>
        </div>

        <div v-if="loadingUsers" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
        <div v-else class="table-wrapper">
          <table class="users-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>{{ tr('Полное имя', 'Толық аты') }}</th>
                <th>{{ tr('Логин', 'Логин') }}</th>
                <th>Email</th>
                <th>{{ tr('Должность', 'Лауазымы') }}</th>
                <th>{{ tr('Роль', 'Рөлі') }}</th>
                <th>{{ tr('Дата создания', 'Құрылған күні') }}</th>
                <th>{{ tr('Действие', 'Әрекет') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in users" :key="item.id">
                <td>{{ item.id }}</td>
                <td>{{ item.full_name || '—' }}</td>
                <td>{{ item.username }}</td>
                <td>{{ item.email || '—' }}</td>
                <td>{{ item.position || '—' }}</td>
                <td>{{ roleLabel(item.role) ?? item.role }}</td>
                <td>{{ formatDate(item.created_at) }}</td>
                <td>
                  <button
                    type="button"
                    class="users-icon-btn users-delete-btn"
                    :disabled="!canDeleteUser(item)"
                    :title="tr('Удалить пользователя', 'Пайдаланушыны өшіру')"
                    @click="openDeleteModal(item)"
                  >
                    <svg viewBox="0 0 24 24" aria-hidden="true">
                      <path d="M9 3.5a1 1 0 0 0-.8.4L7.5 5H4.75a.75.75 0 0 0 0 1.5h.82l.86 11.21A2.5 2.5 0 0 0 8.92 20h6.16a2.5 2.5 0 0 0 2.49-2.29l.86-11.21h.82a.75.75 0 0 0 0-1.5H16.5l-.7-1.1a1 1 0 0 0-.84-.4H9Zm-.99 3h7.98l-.84 10.92a1 1 0 0 1-1 .91H9.85a1 1 0 0 1-1-.91L8.01 6.5ZM10 9.25a.75.75 0 0 0-1.5 0v5.5a.75.75 0 0 0 1.5 0v-5.5Zm4.75-.75a.75.75 0 0 0-.75.75v5.5a.75.75 0 0 0 1.5 0v-5.5a.75.75 0 0 0-.75-.75Zm-2.75.75a.75.75 0 0 0-1.5 0v5.5a.75.75 0 0 0 1.5 0v-5.5Z" />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <template v-else>
      <div class="users-top-grid users-top-grid-requests">
        <section class="panel panel-strong users-stats-card">
          <div class="panel-header">
            <div>
              <h3 class="panel-title">{{ tr('Состояние запросов', 'Сұраныстар күйі') }}</h3>
              <p class="panel-subtitle">{{ tr('Очередь регистрации, решения и уже закрытые заявки.', 'Тіркелу кезегі, шешімдер және жабылған сұраныстар.') }}</p>
            </div>
            <span class="kicker">{{ tr('Запросы', 'Сұраныстар') }}</span>
          </div>

          <div class="users-stats-grid">
            <div class="info-card">
              <span>{{ tr('Всего', 'Барлығы') }}</span>
              <strong>{{ requestStats.total }}</strong>
            </div>
            <div class="info-card">
              <span>{{ tr('Ожидают', 'Күтілуде') }}</span>
              <strong>{{ requestStats.pending }}</strong>
            </div>
            <div class="info-card">
              <span>{{ tr('Одобрены', 'Қабылданды') }}</span>
              <strong>{{ requestStats.approved }}</strong>
            </div>
            <div class="info-card">
              <span>{{ tr('Отклонены', 'Қабылданбады') }}</span>
              <strong>{{ requestStats.rejected }}</strong>
            </div>
          </div>
        </section>
      </div>

      <section class="panel panel-strong users-list-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Запросы для регистрации', 'Тіркелуге арналған сұраныстар') }}</h3>
            <p class="panel-subtitle">{{ tr('Нажмите на иконку действия, чтобы открыть полную карточку и принять решение.', 'Әрекет иконасын басып, толық карточканы ашып, шешім қабылдаңыз.') }}</p>
          </div>
          <span class="kicker">{{ tr('Очередь', 'Кезек') }}</span>
        </div>

        <div v-if="loadingRequests" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
        <div v-else-if="registrationRequests.length === 0" class="empty-state">
          {{ tr('Пока нет запросов на регистрацию.', 'Әзірге тіркелу сұраныстары жоқ.') }}
        </div>
        <div v-else class="users-requests-list">
          <article
            v-for="item in registrationRequests"
            :key="item.id"
            class="users-request-row"
          >
            <div class="users-request-copy">
              <strong>{{ item.full_name || '—' }}</strong>
              <span>{{ item.email || '—' }}</span>
            </div>

            <div class="users-request-meta">
              <span class="users-request-status" :class="`is-${item.status}`">
                {{ requestStatusLabel(item.status) }}
              </span>

              <div class="users-request-actions">
                <button
                  type="button"
                  class="users-icon-btn is-approve"
                  :disabled="item.status !== 'pending'"
                  :title="tr('Принять', 'Қабылдау')"
                  @click="openRequestDecision(item, 'approve')"
                >
                  ✓
                </button>
                <button
                  type="button"
                  class="users-icon-btn is-reject"
                  :disabled="item.status !== 'pending'"
                  :title="tr('Отклонить', 'Қабылдамау')"
                  @click="openRequestDecision(item, 'reject')"
                >
                  ✕
                </button>
              </div>
            </div>
          </article>
        </div>
      </section>
    </template>

    <div v-if="requestModalOpen && selectedRequest" class="modal-backdrop" @click.self="closeRequestModal">
      <div class="modal-card users-request-modal">
        <h3 class="modal-title">
          {{
            requestAction === 'approve'
              ? tr('Подтвердить регистрацию', 'Тіркелуді қабылдау')
              : tr('Отклонить регистрацию', 'Тіркелуді қабылдамау')
          }}
        </h3>
        <p class="modal-subtitle">
          {{
            requestAction === 'approve'
              ? tr('Проверьте карточку пользователя и подтвердите создание аккаунта.', 'Пайдаланушы деректерін тексеріп, аккаунт құруды растаңыз.')
              : tr('Проверьте карточку пользователя и подтвердите отклонение заявки.', 'Пайдаланушы деректерін тексеріп, сұранысты қабылдамауды растаңыз.')
          }}
        </p>

        <div class="users-request-details">
          <div v-for="item in requestDetails" :key="item.label" class="info-card">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>

        <label v-if="requestAction === 'reject'" class="users-reason-field">
          {{ tr('Причина отклонения (опционально)', 'Қабылдамау себебі (міндетті емес)') }}
          <textarea v-model="rejectReason" rows="4"></textarea>
        </label>

        <p class="users-confirm-copy">
          {{
            requestAction === 'approve'
              ? tr('Вы уверены, что хотите принять пользователя?', 'Пайдаланушыны қабылдағыңыз келетініне сенімдісіз бе?')
              : tr('Вы уверены, что хотите отклонить пользователя?', 'Пайдаланушыны қабылдамағыңыз келетініне сенімдісіз бе?')
          }}
        </p>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeRequestModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button
            class="btn"
            :class="requestAction === 'approve' ? 'btn-primary' : 'btn-danger'"
            type="button"
            :disabled="requestActionLoading"
            @click="submitRequestDecision"
          >
            {{
              requestActionLoading
                ? tr('Сохранение...', 'Сақталуда...')
                : requestAction === 'approve'
                  ? tr('Подтвердить', 'Растау')
                  : tr('Отклонить', 'Қабылдамау')
            }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="deleteModalOpen && selectedUser" class="modal-backdrop" @click.self="closeDeleteModal">
      <div class="modal-card users-delete-modal">
        <h3 class="modal-title">{{ tr('Удалить пользователя', 'Пайдаланушыны өшіру') }}</h3>
        <p class="modal-subtitle">
          {{ tr('Аккаунт будет удален из системы. Связанные назначения и отправленные данным пользователем отчеты тоже будут очищены.', 'Аккаунт жүйеден өшіріледі. Осы пайдаланушыға байланған тағайындаулар мен жіберілген есептер де тазартылады.') }}
        </p>

        <div class="users-delete-summary">
          <div class="info-card">
            <span>{{ tr('Полное имя', 'Толық аты') }}</span>
            <strong>{{ selectedUser.full_name || '—' }}</strong>
          </div>
          <div class="info-card">
            <span>{{ tr('Логин', 'Логин') }}</span>
            <strong>{{ selectedUser.username || '—' }}</strong>
          </div>
          <div class="info-card">
            <span>Email</span>
            <strong>{{ selectedUser.email || '—' }}</strong>
          </div>
          <div class="info-card">
            <span>{{ tr('Роль', 'Рөлі') }}</span>
            <strong>{{ roleLabel(selectedUser.role) || '—' }}</strong>
          </div>
        </div>

        <p class="users-confirm-copy">
          {{ tr('Вы уверены, что хотите удалить пользователя?', 'Пайдаланушыны өшіргіңіз келетініне сенімдісіз бе?') }}
        </p>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeDeleteModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button class="btn btn-danger" type="button" :disabled="deleteLoading" @click="submitDeleteUser">
            {{ deleteLoading ? tr('Удаление...', 'Өшірілуде...') : tr('Да', 'Иә') }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.users-tab-strip {
  display: inline-flex;
  gap: 0.5rem;
  padding: 0.35rem;
  margin-bottom: 1rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.74);
}

.users-tab-btn {
  border: 0;
  border-radius: 999px;
  padding: 0.78rem 1.2rem;
  background: transparent;
  color: var(--muted-strong);
  font-weight: 800;
  cursor: pointer;
}

.users-tab-btn.is-active {
  color: var(--text);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(239, 255, 251, 0.88));
  box-shadow: 0 14px 30px rgba(84, 101, 167, 0.12);
}

.users-top-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: minmax(0, 1.3fr) minmax(280px, 0.7fr);
}

.users-top-grid-requests {
  grid-template-columns: minmax(0, 1fr);
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
.users-submit,
.users-full-row {
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

.users-requests-list {
  display: grid;
  gap: 0.9rem;
}

.users-request-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.1rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.7);
}

.users-request-copy {
  display: grid;
  gap: 0.2rem;
}

.users-request-copy span {
  color: var(--muted);
}

.users-request-meta {
  display: flex;
  align-items: center;
  gap: 0.9rem;
}

.users-request-actions {
  display: flex;
  gap: 0.55rem;
}

.users-request-status {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 2.2rem;
  padding: 0.4rem 0.9rem;
  border-radius: 999px;
  font-weight: 800;
  font-size: 0.92rem;
}

.users-request-status.is-pending {
  color: #9a6428;
  background: rgba(255, 229, 188, 0.5);
}

.users-request-status.is-approved {
  color: #166b4f;
  background: rgba(189, 240, 216, 0.5);
}

.users-request-status.is-rejected {
  color: #ab4f48;
  background: rgba(249, 211, 207, 0.55);
}

.users-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.6rem;
  height: 2.6rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.9);
  font-size: 1.15rem;
  font-weight: 900;
  cursor: pointer;
}

.users-icon-btn svg {
  width: 1.1rem;
  height: 1.1rem;
  fill: currentColor;
}

.users-icon-btn.is-approve {
  color: #1c8160;
}

.users-icon-btn.is-reject {
  color: #c7594d;
}

.users-delete-btn {
  color: #c7594d;
}

.users-icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.users-request-modal {
  width: min(780px, calc(100vw - 2rem));
}

.users-delete-modal {
  width: min(620px, calc(100vw - 2rem));
}

.users-request-details {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 1rem;
}

.users-delete-summary {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 1rem;
}

.users-reason-field {
  display: grid;
  gap: 0.45rem;
  margin-bottom: 1rem;
}

.users-reason-field textarea {
  width: 100%;
  min-height: 110px;
  resize: vertical;
}

.users-confirm-copy {
  margin: 0 0 1rem;
  font-weight: 700;
}

@media (max-width: 1080px) {
  .users-top-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 880px) {
  .users-form,
  .users-request-details,
  .users-delete-summary {
    grid-template-columns: 1fr;
  }

  .users-request-row,
  .users-request-meta {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
