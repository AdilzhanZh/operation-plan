<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { logoutRequest } from './services/auth.service'
import { useAuthStore } from './store/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const navigation = [
  { name: 'Dashboard', description: 'Обзор статусов и сроков', to: { name: 'dashboard' } },
  { name: 'Profile', description: 'Аккаунт и настройки доступа', to: { name: 'profile' } },
  { name: 'Users', description: 'Пользователи системы', to: { name: 'users' }, roles: ['admin'] },
  { name: 'Plans', description: 'Рабочие планы по текущему году', to: { name: 'plans' }, roles: ['admin', 'prorector', 'viewer'] },
  { name: 'Planning Period', description: 'Целевые показатели по годам', to: { name: 'planning-period' }, roles: ['admin', 'prorector'] },
  { name: 'Program Execution', description: 'Отчеты и согласование', to: { name: 'program-execution' }, roles: ['admin', 'prorector'] }
]

const roleLabels = {
  admin: 'Administrator',
  prorector: 'Prorector',
  viewer: 'Viewer'
}

const isAuthPage = computed(() => route.name === 'login' || route.name === 'register')
const visibleNavigation = computed(() => {
  const currentRole = authStore.user?.role
  return navigation.filter((item) => !item.roles?.length || item.roles.includes(currentRole))
})

const currentUserName = computed(() => authStore.user?.full_name || authStore.user?.username || 'Oper Plan User')
const currentUserRole = computed(() => roleLabels[authStore.user?.role] ?? authStore.user?.role ?? 'Workspace member')

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
        <span class="sidebar-chip">Operational Planning Platform</span>
        <h1 class="brand">Oper Plan</h1>
        <p class="subtitle">Korkyt Ata University workspace for planning, execution and review.</p>
      </div>

      <section class="sidebar-profile">
        <span class="sidebar-label">Текущий аккаунт</span>
        <strong>{{ currentUserName }}</strong>
        <span>{{ currentUserRole }}</span>
      </section>

      <nav class="sidebar-nav" aria-label="Основная навигация">
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
        <p class="sidebar-note">Единая среда для сроков, ответственных и контрольных отчетов.</p>
        <button type="button" class="btn btn-secondary sidebar-logout" @click="logout">Выйти</button>
      </div>
    </aside>

    <main class="app-main">
      <div class="app-main-inner">
        <RouterView />
      </div>
    </main>
  </div>
</template>
