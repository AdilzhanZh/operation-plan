<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LanguageSwitch from './components/LanguageSwitch.vue'
import { useLocale } from './composables/useLocale'
import { logoutRequest } from './services/auth.service'
import { useAuthStore } from './store/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { tr } = useLocale()

const navigation = computed(() => [
  { name: tr('Панель управления', 'Басқару панелі'), description: tr('Обзор статусов и сроков', 'Мәртебелер мен мерзімдер шолуы'), to: { name: 'dashboard' } },
  { name: tr('Профиль', 'Профиль'), description: tr('Аккаунт и настройки доступа', 'Аккаунт пен қолжетімділік баптаулары'), to: { name: 'profile' } },
  { name: tr('Пользователи', 'Пайдаланушылар'), description: tr('Пользователи системы', 'Жүйе пайдаланушылары'), to: { name: 'users' }, roles: ['admin'] },
  { name: tr('Планы и отчеты', 'Жоспарлар мен есептер'), description: tr('Рабочие планы и история отчетов по текущему году', 'Ағымдағы жылға арналған жұмыс жоспарлары мен есеп тарихы'), to: { name: 'plans' }, roles: ['admin', 'prorector', 'viewer'] },
  { name: tr('Плановый период', 'Жоспарлы кезең'), description: tr('Целевые показатели по годам', 'Жылдар бойынша мақсатты индикаторлар'), to: { name: 'planning-period' }, roles: ['admin', 'prorector'] },
  { name: tr('Выполнение программы', 'Бағдарлама орындалуы'), description: tr('Отчеты и согласование', 'Есептер және келісу'), to: { name: 'program-execution' }, roles: ['admin', 'prorector'] }
])

const isAuthPage = computed(() => route.name === 'login' || route.name === 'register')
const visibleNavigation = computed(() => {
  const currentRole = authStore.user?.role
  return navigation.value.filter((item) => !item.roles?.length || item.roles.includes(currentRole))
})

const currentUserName = computed(() => authStore.user?.full_name || authStore.user?.username || tr('Пользователь Oper Plan', 'Oper Plan пайдаланушысы'))
const currentUserRole = computed(() => {
  const role = String(authStore.user?.role ?? '')
  if (role === 'admin') {
    return tr('Администратор', 'Әкімші')
  }
  if (role === 'prorector') {
    return tr('Проректор', 'Проректор')
  }
  if (role === 'viewer') {
    return tr('Наблюдатель', 'Бақылаушы')
  }
  return role || tr('Участник рабочей области', 'Жұмыс кеңістігінің қатысушысы')
})

async function logout() {
  try {
    await logoutRequest()
  } catch {
    // Local cleanup is still required when token is already expired.
  } finally {
    authStore.logout()
    router.push({ name: 'login' })
  }
}
</script>

<template>
  <div v-if="isAuthPage" class="auth-layout">
    <RouterView />
  </div>

  <div v-else class="app-shell">
    <aside class="app-sidebar">
      <div class="sidebar-top">
        <span class="sidebar-chip">{{ tr('Платформа операционного планирования', 'Операциялық жоспарлау платформасы') }}</span>
        <h1 class="brand">Oper Plan</h1>
        <p class="subtitle">{{ tr('Рабочее пространство университета Коркыт Ата для планирования, исполнения и контроля.', 'Қорқыт Ата университетінің жоспарлау, орындау және бақылау жұмыс кеңістігі.') }}</p>
      </div>

      <section class="sidebar-profile">
        <span class="sidebar-label">{{ tr('Текущий аккаунт', 'Ағымдағы аккаунт') }}</span>
        <strong>{{ currentUserName }}</strong>
        <span>{{ currentUserRole }}</span>
        <div class="sidebar-lang">
          <LanguageSwitch />
        </div>
      </section>

      <nav class="sidebar-nav" :aria-label="tr('Основная навигация', 'Негізгі навигация')">
        <RouterLink
          v-for="item in visibleNavigation"
          :key="item.name"
          :to="item.to"
          class="menu-item"
        >
          <span class="menu-item-title">{{ item.name }}</span>
          <span class="menu-item-text">{{ item.description }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <p class="sidebar-note">{{ tr('Единая среда для сроков, ответственных и контрольных отчетов.', 'Мерзімдер, жауаптылар және бақылау есептері үшін бірыңғай орта.') }}</p>
        <button type="button" class="btn btn-secondary sidebar-logout" @click="logout">{{ tr('Выйти', 'Шығу') }}</button>
      </div>
    </aside>

    <main class="app-main">
      <div class="app-main-inner">
        <RouterView />
      </div>
    </main>
  </div>
</template>
