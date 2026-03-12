<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import LanguageSwitch from '../components/LanguageSwitch.vue'
import { useLocale } from '../composables/useLocale'
import { registerRequest } from '../services/auth.service'

const router = useRouter()
const { tr } = useLocale()

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
    successMessage.value = tr('Регистрация успешно завершена. Теперь войдите в систему.', 'Тіркелу сәтті аяқталды. Енді жүйеге кіріңіз.')
    setTimeout(() => {
      router.push({ name: 'login' })
    }, 700)
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr('Ошибка при регистрации', 'Тіркелу кезінде қате болды')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="auth-card">
    <div class="auth-showcase">
      <span class="auth-kicker">{{ tr('Новый аккаунт', 'Жаңа аккаунт') }}</span>
      <h1 class="auth-title">{{ tr('Подключите нового участника к рабочему циклу.', 'Жұмыс цикліне жаңа қатысушыны қосыңыз.') }}</h1>
      <p class="auth-lead">
        {{ tr('После регистрации пользователь сможет видеть планы, сроки и статусы в рамках своей роли.', 'Тіркелгеннен кейін пайдаланушы өз рөліне сай жоспарлар, мерзімдер және мәртебелерді көреді.') }}
      </p>

      <div class="auth-points">
        <div class="auth-point">
          <strong>{{ tr('Быстрое подключение', 'Жылдам қосылу') }}</strong>
          <span>{{ tr('Новый аккаунт сразу получает базовую роль `viewer` и доступ к просмотру ключевых разделов.', 'Жаңа аккаунт бірден `viewer` базалық рөлін алып, негізгі бөлімдерді көруге қолжеткізеді.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Чистая структура данных', 'Таза дерек құрылымы') }}</strong>
          <span>{{ tr('Имя, фамилия и логин собираются в одном шаге без лишних переходов.', 'Аты-жөні мен логин артық қадамсыз бір әрекетте толтырылады.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Готовность к маршруту согласования', 'Келісу маршрутына дайын') }}</strong>
          <span>{{ tr('После создания аккаунт можно перевести в нужную роль через раздел пользователей.', 'Аккаунт құрылғаннан кейін пайдаланушылар бөлімінде қажетті рөлге ауыстыруға болады.') }}</span>
        </div>
      </div>
    </div>

    <div class="auth-form-panel">
      <div class="auth-lang-row">
        <LanguageSwitch />
      </div>
      <span class="kicker">{{ tr('Регистрация', 'Тіркелу') }}</span>
      <h1>{{ tr('Создание аккаунта', 'Аккаунт құру') }}</h1>
      <p>{{ tr('Заполните карточку пользователя. После успешной регистрации откроется страница входа.', 'Пайдаланушы картасын толтырыңыз. Сәтті тіркелуден кейін кіру беті ашылады.') }}</p>

      <form class="auth-actions" @submit.prevent="submit">
        <div class="auth-form-grid columns-2">
          <label>
            {{ tr('Имя', 'Аты') }}
            <input v-model="form.first_name" type="text" autocomplete="given-name" required />
          </label>

          <label>
            {{ tr('Фамилия', 'Фамилиясы') }}
            <input v-model="form.last_name" type="text" autocomplete="family-name" required />
          </label>

          <label class="full-row">
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
        </div>

        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

        <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
          {{ loading ? tr('Регистрация...', 'Тіркелу...') : tr('Зарегистрироваться', 'Тіркелу') }}
        </button>
      </form>

      <p class="auth-link-row">
        {{ tr('Уже есть аккаунт?', 'Аккаунт бар ма?') }}
        <RouterLink :to="{ name: 'login' }">{{ tr('Вход', 'Кіру') }}</RouterLink>
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

.full-row {
  grid-column: 1 / -1;
}

@media (max-width: 920px) {
  .full-row {
    grid-column: auto;
  }
}
</style>
