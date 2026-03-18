<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LanguageSwitch from './components/LanguageSwitch.vue'
import { useLocale } from './composables/useLocale'
import { logoutRequest } from './services/auth.service'
import { useAuthStore } from './store/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { tr } = useLocale()
const profileMenuOpen = ref(false)
const profileMenuRef = ref(null)

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
const currentUserInitials = computed(() => {
  const parts = currentUserName.value
    .split(/\s+/)
    .map((part) => part.trim())
    .filter(Boolean)

  if (parts.length === 0) {
    return 'OP'
  }

  return parts
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('')
})

function closeProfileMenu() {
  profileMenuOpen.value = false
}

function toggleProfileMenu() {
  profileMenuOpen.value = !profileMenuOpen.value
}

function handleProfileMenuClickOutside(event) {
  const menuElement = profileMenuRef.value
  if (menuElement && !menuElement.contains(event.target)) {
    closeProfileMenu()
  }
}

function handleProfileMenuKeydown(event) {
  if (event.key === 'Escape') {
    closeProfileMenu()
  }
}

async function logout() {
  try {
    await logoutRequest()
  } catch {
    // Local cleanup is still required when token is already expired.
  } finally {
    closeProfileMenu()
    authStore.logout()
    router.push({ name: 'login' })
  }
}

onMounted(() => {
  document.addEventListener('click', handleProfileMenuClickOutside)
  document.addEventListener('keydown', handleProfileMenuKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleProfileMenuClickOutside)
  document.removeEventListener('keydown', handleProfileMenuKeydown)
})
</script>

<template>
  <div v-if="isAuthPage" class="auth-layout">
    <RouterView />
  </div>

  <div v-else class="app-shell">
    <div class="app-frame">
      <header class="app-topbar">
        <div class="topbar-brand-row">
          <RouterLink :to="{ name: 'dashboard' }" class="topbar-brand">
            <span class="topbar-brand-mark">OP</span>
            <span class="topbar-brand-copy">
              <strong>Oper Plan</strong>
              <span>{{ tr('Панель координации и контроля', 'Үйлестіру және бақылау панелі') }}</span>
            </span>
          </RouterLink>
        </div>

        <nav class="topbar-nav" :aria-label="tr('Основная навигация', 'Негізгі навигация')">
          <RouterLink
            v-for="item in visibleNavigation"
            :key="item.name"
            :to="item.to"
            class="topbar-nav-link"
          >
            {{ item.name }}
          </RouterLink>
        </nav>

        <div ref="profileMenuRef" class="topbar-meta">
          <button
            type="button"
            class="unstyled-button topbar-avatar-btn"
            :class="{ 'is-open': profileMenuOpen }"
            :aria-expanded="profileMenuOpen ? 'true' : 'false'"
            :aria-label="tr('Открыть меню профиля', 'Профиль мәзірін ашу')"
            @click="toggleProfileMenu"
          >
            <span class="topbar-avatar">{{ currentUserInitials }}</span>
          </button>

          <div v-if="profileMenuOpen" class="topbar-profile-menu">
            <div class="topbar-profile-card">
              <div class="topbar-user-copy">
                <strong>{{ currentUserName }}</strong>
                <span>{{ currentUserRole }}</span>
              </div>
              <span class="topbar-avatar topbar-avatar-static">{{ currentUserInitials }}</span>
            </div>

            <RouterLink :to="{ name: 'profile' }" class="btn btn-ghost topbar-profile-link" @click="closeProfileMenu">
              {{ tr('Открыть профиль', 'Профильді ашу') }}
            </RouterLink>

            <div class="topbar-profile-section">
              <span class="topbar-profile-label">{{ tr('Язык интерфейса', 'Интерфейс тілі') }}</span>
              <LanguageSwitch />
            </div>

            <button type="button" class="btn btn-secondary topbar-menu-logout" @click="logout">
              {{ tr('Выйти', 'Шығу') }}
            </button>
          </div>
        </div>
      </header>

      <main class="app-main">
        <div class="app-main-inner">
          <RouterView />
        </div>
      </main>
    </div>
  </div>
</template>
