<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './store/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const navigation = [
  { name: 'Dashboard', to: { name: 'dashboard' } },
  { name: 'Profile', to: { name: 'profile' } },
  { name: 'Users', to: { name: 'users' }, roles: ['admin'] },
  { name: 'Plans', to: { name: 'plans' }, roles: ['admin', 'prorector', 'viewer'] },
  { name: 'Planning Period', to: { name: 'planning-period' }, roles: ['admin', 'prorector'] },
  { name: 'Выполнение Программы Развития', to: { name: 'program-execution' }, roles: ['admin', 'prorector'] }
]

const isAuthPage = computed(() => route.name === 'login' || route.name === 'register')
const visibleNavigation = computed(() => {
  const currentRole = authStore.user?.role
  return navigation.filter((item) => !item.roles?.length || item.roles.includes(currentRole))
})

function logout() {
  authStore.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div v-if="isAuthPage" class="auth-layout">
    <RouterView />
  </div>

  <div v-else class="app-layout">
    <aside class="sidebar">
      <h1 class="brand">Oper Plan</h1>
      <p class="subtitle">Korkyt Ata University</p>

      <nav class="menu">
        <RouterLink
          v-for="item in visibleNavigation"
          :key="item.name"
          :to="item.to"
          class="menu-item"
        >
          {{ item.name }}
        </RouterLink>
      </nav>

      <button type="button" class="logout" @click="logout">Logout</button>
    </aside>

    <main class="content">
      <RouterView />
    </main>
  </div>
</template>
