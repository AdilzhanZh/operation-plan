<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import LanguageSwitch from '../components/LanguageSwitch.vue'
import { useLocale } from '../composables/useLocale'
import {
  completePasswordResetRequest,
  requestPasswordResetCode,
  verifyPasswordResetCode
} from '../services/auth.service'

const router = useRouter()
const { tr } = useLocale()

const step = ref(1)
const loading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const resetToken = ref('')

const emailForm = reactive({
  email: ''
})

const codeForm = reactive({
  code: ''
})

const passwordForm = reactive({
  new_password: '',
  confirm_password: ''
})

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

async function submitEmail() {
  loading.value = true
  clearMessages()

  try {
    const response = await requestPasswordResetCode({ email: emailForm.email })
    successMessage.value = response?.message ?? tr(
      'Код отправлен на email. Введите его на следующем шаге.',
      'Код email-ға жіберілді. Келесі қадамда оны енгізіңіз.'
    )
    step.value = 2
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка при отправке кода',
      'Код жіберу кезінде қате болды'
    )
  } finally {
    loading.value = false
  }
}

async function submitCode() {
  loading.value = true
  clearMessages()

  try {
    const response = await verifyPasswordResetCode({
      email: emailForm.email,
      code: codeForm.code
    })
    resetToken.value = response?.reset_token ?? ''
    successMessage.value = tr(
      'Код подтвержден. Теперь задайте новый пароль.',
      'Код расталды. Енді жаңа құпиясөз орнатыңыз.'
    )
    step.value = 3
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Неверный код подтверждения',
      'Растау коды қате'
    )
  } finally {
    loading.value = false
  }
}

async function submitPassword() {
  loading.value = true
  clearMessages()

  try {
    await completePasswordResetRequest({
      reset_token: resetToken.value,
      new_password: passwordForm.new_password,
      confirm_password: passwordForm.confirm_password
    })
    successMessage.value = tr(
      'Пароль обновлен. Теперь войдите в систему.',
      'Құпиясөз жаңартылды. Енді жүйеге кіріңіз.'
    )
    step.value = 4
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка при обновлении пароля',
      'Құпиясөзді жаңарту кезінде қате болды'
    )
  } finally {
    loading.value = false
  }
}

function goToStep(nextStep) {
  clearMessages()
  step.value = nextStep
}

async function goToLogin() {
  await router.push({ name: 'login', query: { reauth: '1' } })
}
</script>

<template>
  <section class="auth-card">
    <div class="auth-showcase">
      <span class="auth-kicker">{{ tr('Восстановление доступа', 'Қолжетімділікті қалпына келтіру') }}</span>
      <h1 class="auth-title">
        {{ tr('Смена пароля через email-подтверждение.', 'Құпиясөзді email арқылы растаумен өзгерту.') }}
      </h1>
      <p class="auth-lead">
        {{ tr('Система проверит email, отправит шестизначный код и только после подтверждения даст задать новый пароль.', 'Жүйе email-ды тексеріп, алты таңбалы код жібереді және содан кейін ғана жаңа құпиясөз орнатуға мүмкіндік береді.') }}
      </p>

      <div class="auth-points">
        <div class="auth-point">
          <strong>{{ tr('Шаг 1', '1-қадам') }}</strong>
          <span>{{ tr('Укажите email аккаунта, который уже есть в базе.', 'Базада бар аккаунттың email-ын енгізіңіз.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Шаг 2', '2-қадам') }}</strong>
          <span>{{ tr('Подтвердите шестизначный код из письма.', 'Хаттағы алты таңбалы кодты растаңыз.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Шаг 3', '3-қадам') }}</strong>
          <span>{{ tr('Назначьте новый пароль с проверкой повтора.', 'Жаңа құпиясөзді қайталап енгізіп орнатыңыз.') }}</span>
        </div>
      </div>
    </div>

    <div class="auth-form-panel">
      <div class="auth-lang-row">
        <LanguageSwitch />
      </div>
      <span class="kicker">{{ tr('Забыли пароль', 'Құпиясөз ұмытылды') }}</span>
      <h1>{{ tr('Восстановление пароля', 'Құпиясөзді қалпына келтіру') }}</h1>
      <p>
        {{
          step === 1
            ? tr('Введите email, привязанный к вашему аккаунту.', 'Аккаунтыңызға тіркелген email-ды енгізіңіз.')
            : step === 2
              ? tr('Введите шестизначный код, который был отправлен на почту.', 'Поштаға жіберілген алты таңбалы кодты енгізіңіз.')
              : step === 3
                ? tr('Придумайте новый пароль и повторите его.', 'Жаңа құпиясөз ойлап тауып, оны қайталап енгізіңіз.')
                : tr('Пароль уже обновлен. Можно вернуться ко входу.', 'Құпиясөз жаңартылды. Енді кіру бетіне оралуға болады.')
        }}
      </p>

      <form v-if="step === 1" class="auth-actions" @submit.prevent="submitEmail">
        <div class="auth-form-grid">
          <label>
            Email
            <input v-model="emailForm.email" type="email" autocomplete="email" required />
          </label>
        </div>

        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

        <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
          {{ loading ? tr('Отправка...', 'Жіберілуде...') : tr('Отправить код', 'Код жіберу') }}
        </button>
      </form>

      <form v-else-if="step === 2" class="auth-actions" @submit.prevent="submitCode">
        <div class="auth-form-grid">
          <label>
            {{ tr('Код подтверждения', 'Растау коды') }}
            <input v-model="codeForm.code" type="text" inputmode="numeric" maxlength="6" required />
          </label>
        </div>

        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

        <div class="auth-row-actions">
          <button type="button" class="btn btn-ghost" @click="goToStep(1)">
            {{ tr('Назад', 'Артқа') }}
          </button>
          <button type="submit" class="btn btn-primary" :disabled="loading">
            {{ loading ? tr('Проверка...', 'Тексерілуде...') : tr('Подтвердить код', 'Кодты растау') }}
          </button>
        </div>
      </form>

      <form v-else-if="step === 3" class="auth-actions" @submit.prevent="submitPassword">
        <div class="auth-form-grid">
          <label>
            {{ tr('Новый пароль', 'Жаңа құпиясөз') }}
            <input v-model="passwordForm.new_password" type="password" autocomplete="new-password" required />
          </label>
          <label>
            {{ tr('Новый пароль (повтор)', 'Жаңа құпиясөз (қайта)') }}
            <input v-model="passwordForm.confirm_password" type="password" autocomplete="new-password" required />
          </label>
        </div>

        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

        <button type="submit" class="btn btn-primary auth-submit" :disabled="loading">
          {{ loading ? tr('Сохранение...', 'Сақталуда...') : tr('Обновить пароль', 'Құпиясөзді жаңарту') }}
        </button>
      </form>

      <div v-else class="auth-actions">
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>
        <button type="button" class="btn btn-primary auth-submit" @click="goToLogin">
          {{ tr('Вернуться ко входу', 'Кіру бетіне оралу') }}
        </button>
      </div>

      <p class="auth-link-row">
        {{ tr('Вспомнили пароль?', 'Құпиясөз есіңізге түсті ме?') }}
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

.auth-row-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 0.75rem;
}
</style>
