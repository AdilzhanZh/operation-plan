<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LanguageSwitch from '../components/LanguageSwitch.vue'
import { useLocale } from '../composables/useLocale'
import { loginRequest } from '../services/auth.service'
import { useAuthStore } from '../store/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { tr } = useLocale()

const form = reactive({
  username: '',
  password: ''
})

const loading = ref(false)
const errorMessage = ref('')
const infoMessage = ref(route.query.reauth === '1'
  ? tr('Сессия завершена. Войдите снова с новым паролем.', 'Сессия аяқталды. Жаңа құпиясөзбен қайта кіріңіз.')
  : '')

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
    errorMessage.value = error?.response?.data?.error ?? tr('Ошибка при входе', 'Кіру кезінде қате болды')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="auth-card">
    <div class="auth-showcase">
      <span class="auth-kicker">Oper Plan</span>
      <h1 class="auth-title">{{ tr('Планирование, контроль и отчеты в одном контуре.', 'Жоспарлау, бақылау және есептер бір ортада.') }}</h1>
      <p class="auth-lead">
        {{ tr('Вход открывает доступ к срокам, ответственным и статусам выполнения по всей программе развития.', 'Кіру бағдарламаның мерзімдері, жауаптылары және орындалу мәртебелеріне қолжеткізеді.') }}
      </p>

      <div class="auth-points">
        <div class="auth-point">
          <strong>{{ tr('Единый рабочий контур', 'Бірыңғай жұмыс контуры') }}</strong>
          <span>{{ tr('Все ключевые показатели, планы и отчеты собраны в одном интерфейсе.', 'Барлық негізгі көрсеткіштер, жоспарлар мен есептер бір интерфейсте жинақталған.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Прозрачные сроки', 'Айқын мерзімдер') }}</strong>
          <span>{{ tr('Ответственные, дедлайны и статусы видны без ручного согласования в таблицах.', 'Жауаптылар, дедлайндар мен мәртебелер кестеде бірден көрінеді.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Быстрый контроль', 'Жедел бақылау') }}</strong>
          <span>{{ tr('Администраторы и проректоры работают в одном процессе без переключения между системами.', 'Әкімшілер мен проректорлар жүйелер арасында ауыспай, бір процесте жұмыс істейді.') }}</span>
        </div>
      </div>
    </div>

    <div class="auth-form-panel">
      <div class="auth-lang-row">
        <LanguageSwitch />
      </div>
      <span class="kicker">{{ tr('Вход', 'Кіру') }}</span>
      <h1>{{ tr('Вход в систему', 'Жүйеге кіру') }}</h1>
      <p>{{ tr('Используйте логин и пароль, выданные администратором платформы.', 'Платформа әкімшісі берген логин мен құпиясөзді пайдаланыңыз.') }}</p>

      <form class="auth-actions" @submit.prevent="submit">
        <div class="auth-form-grid">
          <label>
            {{ tr('Логин', 'Логин') }}
            <input v-model="form.username" type="text" autocomplete="username" required />
          </label>

          <label>
            {{ tr('Пароль', 'Құпиясөз') }}
            <input v-model="form.password" type="password" autocomplete="current-password" required />
          </label>
        </div>

        <p v-if="infoMessage" class="message message-success">{{ infoMessage }}</p>
        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>

        <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
          {{ loading ? tr('Вход...', 'Кіру...') : tr('Войти', 'Кіру') }}
        </button>
      </form>

      <p class="auth-link-row">
        {{ tr('Нет аккаунта?', 'Аккаунтыңыз жоқ па?') }}
        <RouterLink :to="{ name: 'register' }">{{ tr('Регистрация', 'Тіркелу') }}</RouterLink>
      </p>
    </div>
  </section>
</template>

<style scoped>
.auth-lang-row {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 0.5rem;
}

.auth-submit {
  width: 100%;
  justify-content: center;
}
</style>
