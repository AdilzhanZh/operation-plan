<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { registerRequest } from '../services/auth.service'

const router = useRouter()

const form = reactive({
  first_name: '',
  last_name: '',
  middle_name: '',
  username: '',
  password: '',
  confirm_password: ''
})

const loading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

async function submit() {
  loading.value = true
  errorMessage.value = ''
  successMessage.value = ''

  try {
    await registerRequest(form)
    successMessage.value = 'Тіркелу сәтті аяқталды. Енді жүйеге кіріңіз.'
    setTimeout(() => {
      router.push({ name: 'login' })
    }, 700)
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? 'Тіркелу кезінде қате болды'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="auth-card">
    <div class="auth-showcase">
      <span class="auth-kicker">New Account</span>
      <h1 class="auth-title">Подключите нового участника к рабочему циклу.</h1>
      <p class="auth-lead">
        После регистрации пользователь сможет видеть планы, сроки и статусы в рамках своей роли.
      </p>

      <div class="auth-points">
        <div class="auth-point">
          <strong>Быстрое подключение</strong>
          <span>Новый аккаунт сразу получает базовую роль `viewer` и доступ к просмотру ключевых разделов.</span>
        </div>
        <div class="auth-point">
          <strong>Чистая структура данных</strong>
          <span>Имя, фамилия и логин собираются в одном шаге без лишних переходов.</span>
        </div>
        <div class="auth-point">
          <strong>Готовность к маршруту согласования</strong>
          <span>После создания аккаунт можно перевести в нужную роль через раздел пользователей.</span>
        </div>
      </div>
    </div>

    <div class="auth-form-panel">
      <span class="kicker">Registration</span>
      <h1>Создание аккаунта</h1>
      <p>Заполните карточку пользователя. После успешной регистрации откроется страница входа.</p>

      <form class="auth-actions" @submit.prevent="submit">
        <div class="auth-form-grid columns-2">
          <label>
            Аты
            <input v-model="form.first_name" type="text" autocomplete="given-name" required />
          </label>

          <label>
            Фамилиясы
            <input v-model="form.last_name" type="text" autocomplete="family-name" required />
          </label>

          <label class="full-row">
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
        </div>

        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

        <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
          {{ loading ? 'Тіркелу...' : 'Тіркелу' }}
        </button>
      </form>

      <p class="auth-link-row">
        Аккаунт бар ма?
        <RouterLink :to="{ name: 'login' }">Кіру</RouterLink>
      </p>
    </div>
  </section>
</template>

<style scoped>
.auth-submit {
  width: 100%;
  justify-content: center;
}

.full-row {
  grid-column: 1 / -1;
}

@media (max-width: 920px) {
  .full-row {
    grid-column: auto;
  }
}
</style>
