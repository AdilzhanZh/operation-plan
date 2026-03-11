<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { loginRequest } from '../services/auth.service'
import { useAuthStore } from '../store/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const form = reactive({
  username: '',
  password: ''
})

const loading = ref(false)
const errorMessage = ref('')
const infoMessage = ref(route.query.reauth === '1' ? 'Сессия завершена. Войдите снова с новым паролем.' : '')

async function submit() {
  loading.value = true
  errorMessage.value = ''

  try {
    const response = await loginRequest({
      username: form.username,
      password: form.password
    })

    authStore.login(response)
    await router.push({ name: 'dashboard' })
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? 'Кіру кезінде қате болды'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="auth-card">
    <div class="auth-showcase">
      <span class="auth-kicker">Oper Plan</span>
      <h1 class="auth-title">Планирование, контроль и отчеты в одном контуре.</h1>
      <p class="auth-lead">
        Вход открывает доступ к срокам, ответственным и статусам выполнения по всей программе развития.
      </p>

      <div class="auth-points">
        <div class="auth-point">
          <strong>Единый рабочий контур</strong>
          <span>Все ключевые показатели, планы и отчеты собраны в одном интерфейсе.</span>
        </div>
        <div class="auth-point">
          <strong>Прозрачные сроки</strong>
          <span>Ответственные, дедлайны и статусы видны без ручного согласования в таблицах.</span>
        </div>
        <div class="auth-point">
          <strong>Быстрый контроль</strong>
          <span>Администраторы и проректоры работают в одном процессе без переключения между системами.</span>
        </div>
      </div>
    </div>

    <div class="auth-form-panel">
      <span class="kicker">Sign In</span>
      <h1>Вход в систему</h1>
      <p>Используйте логин и пароль, выданные администратором платформы.</p>

      <form class="auth-actions" @submit.prevent="submit">
        <div class="auth-form-grid">
          <label>
            Логин
            <input v-model="form.username" type="text" autocomplete="username" required />
          </label>

          <label>
            Пароль
            <input v-model="form.password" type="password" autocomplete="current-password" required />
          </label>
        </div>

        <p v-if="infoMessage" class="message message-success">{{ infoMessage }}</p>
        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>

        <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
          {{ loading ? 'Кіру...' : 'Кіру' }}
        </button>
      </form>

      <p class="auth-link-row">
        Аккаунтыңыз жоқ па?
        <RouterLink :to="{ name: 'register' }">Тіркелу</RouterLink>
      </p>
    </div>
  </section>
</template>

<style scoped>
.auth-submit {
  width: 100%;
  justify-content: center;
}
</style>
