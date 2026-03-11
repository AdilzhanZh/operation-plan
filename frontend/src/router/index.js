import { createRouter, createWebHistory } from 'vue-router'
import DashboardPage from '../pages/DashboardPage.vue'
import LoginPage from '../pages/LoginPage.vue'
import RegisterPage from '../pages/RegisterPage.vue'
import PlanningPeriodPage from '../pages/PlanningPeriodPage.vue'
import PlansPage from '../pages/PlansPage.vue'
import ProfilePage from '../pages/ProfilePage.vue'
import ProgramExecutionPage from '../pages/ProgramExecutionPage.vue'
import UsersPage from '../pages/UsersPage.vue'
import NotFoundPage from '../pages/NotFoundPage.vue'
import { useAuthStore } from '../store/auth'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: LoginPage,
    meta: { public: true }
  },
  {
    path: '/register',
    name: 'register',
    component: RegisterPage,
    meta: { public: true }
  },
  {
    path: '/',
    redirect: { name: 'dashboard' }
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: DashboardPage,
    meta: { requiresAuth: true }
  },
  {
    path: '/profile',
    name: 'profile',
    component: ProfilePage,
    meta: { requiresAuth: true }
  },
  {
    path: '/users',
    name: 'users',
    component: UsersPage,
    meta: { requiresAuth: true, roles: ['admin'] }
  },
  {
    path: '/plans',
    name: 'plans',
    component: PlansPage,
    meta: { requiresAuth: true, roles: ['admin', 'prorector', 'viewer'] }
  },
  {
    path: '/planning-period',
    name: 'planning-period',
    component: PlanningPeriodPage,
    meta: { requiresAuth: true, roles: ['admin', 'prorector'] }
  },
  {
    path: '/program-execution',
    name: 'program-execution',
    component: ProgramExecutionPage,
    meta: { requiresAuth: true, roles: ['admin', 'prorector'] }
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: NotFoundPage
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    return { name: 'login' }
  }

  if ((to.name === 'login' || to.name === 'register') && authStore.isAuthenticated) {
    return { name: 'dashboard' }
  }

  if (to.meta.roles?.length) {
    const userRole = authStore.user?.role
    if (!userRole || !to.meta.roles.includes(userRole)) {
      return { name: 'dashboard' }
    }
  }

  return true
})

export default router
