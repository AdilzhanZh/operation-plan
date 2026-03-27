<script setup>
import { reactive, ref } from 'vue'
import LanguageSwitch from '../components/LanguageSwitch.vue'
import { useLocale } from '../composables/useLocale'
import {
  requestRegistrationCode,
  verifyRegistrationCode
} from '../services/auth.service'

const { tr } = useLocale()

const step = ref(1)
const loading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

const form = reactive({
  first_name: '',
  last_name: '',
  middle_name: '',
  position: '',
  email: '',
  username: '',
  password: '',
  confirm_password: ''
})

const codeForm = reactive({
  code: ''
})

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

async function submitRegistrationCard() {
  loading.value = true
  clearMessages()

  try {
    const response = await requestRegistrationCode(form)
    successMessage.value = response?.message ?? tr(
      'На email отправлен шестизначный код. Введите его на следующем шаге.',
      'Email-ға алты таңбалы код жіберілді. Келесі қадамда оны енгізіңіз.'
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

async function submitVerificationCode() {
  loading.value = true
  clearMessages()

  try {
    const response = await verifyRegistrationCode({
      email: form.email,
      code: codeForm.code
    })
    successMessage.value = response?.message ?? tr(
      'Запрос на регистрацию отправлен. Дождитесь решения администратора.',
      'Тіркелу сұранысы жіберілді. Әкімші шешімін күтіңіз.'
    )
    step.value = 3
  } catch (error) {
    errorMessage.value = error?.response?.data?.error ?? tr(
      'Ошибка подтверждения кода',
      'Кодты растау кезінде қате болды'
    )
  } finally {
    loading.value = false
  }
}

function returnToCard() {
  clearMessages()
  step.value = 1
}
</script>

<template>
  <section class="auth-card">
    <div class="auth-showcase">
      <span class="auth-kicker">{{ tr('Новый аккаунт', 'Жаңа аккаунт') }}</span>
      <h1 class="auth-title">
        {{ tr('Регистрация теперь проходит через email-подтверждение и согласование администратора.', 'Тіркелу енді email-растау және әкімші мақұлдауы арқылы өтеді.') }}
      </h1>
      <p class="auth-lead">
        {{ tr('Пользователь заполняет карточку, подтверждает email шестизначным кодом и только после этого попадает в очередь на утверждение.', 'Пайдаланушы деректерін толтырады, email-ды алты таңбалы кодпен растайды және содан кейін ғана мақұлдау кезегіне түседі.') }}
      </p>

      <div class="auth-points">
        <div class="auth-point">
          <strong>{{ tr('Email-подтверждение', 'Email-растау') }}</strong>
          <span>{{ tr('Некорректный email не пропускается к следующему шагу.', 'Қате email келесі қадамға өткізілмейді.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Очередь на согласование', 'Келісу кезегі') }}</strong>
          <span>{{ tr('После проверки кода система создает не пользователя, а запрос для администратора.', 'Код расталғаннан кейін жүйе бірден пайдаланушы емес, әкімшіге арналған сұраныс жасайды.') }}</span>
        </div>
        <div class="auth-point">
          <strong>{{ tr('Контроль доступа', 'Қолжетімділікті бақылау') }}</strong>
          <span>{{ tr('До одобрения администратором вход в систему не открывается.', 'Әкімші мақұлдамайынша жүйеге кіру ашылмайды.') }}</span>
        </div>
      </div>
    </div>

    <div class="auth-form-panel">
      <div class="auth-lang-row">
        <LanguageSwitch />
      </div>
      <span class="kicker">{{ tr('Регистрация', 'Тіркелу') }}</span>
      <h1>{{ tr('Запрос на регистрацию', 'Тіркелу сұранысы') }}</h1>
      <p>
        {{
          step === 1
            ? tr('Сначала заполните все поля карточки пользователя.', 'Алдымен пайдаланушы карточкасының барлық өрістерін толтырыңыз.')
            : step === 2
              ? tr('Введите код из email. После проверки запрос уйдет администратору.', 'Email-дан келген кодты енгізіңіз. Тексерістен кейін сұраныс әкімшіге жіберіледі.')
              : tr('Запрос отправлен. Теперь дождитесь решения администратора.', 'Сұраныс жіберілді. Енді әкімшінің шешімін күтіңіз.')
        }}
      </p>

      <form v-if="step === 1" class="auth-actions" @submit.prevent="submitRegistrationCard">
        <div class="auth-form-grid columns-2">
          <label>
            {{ tr('Имя', 'Аты') }}
            <input v-model="form.first_name" type="text" autocomplete="given-name" required />
          </label>

          <label>
            {{ tr('Фамилия', 'Фамилиясы') }}
            <input v-model="form.last_name" type="text" autocomplete="family-name" required />
          </label>

          <label>
            {{ tr('Отчество', 'Тегі') }}
            <input v-model="form.middle_name" type="text" autocomplete="additional-name" />
          </label>

          <label>
            {{ tr('Должность', 'Лауазымы') }}
            <input v-model="form.position" type="text" autocomplete="organization-title" required />
          </label>

          <label class="full-row">
            Email (Gmail)
            <input v-model="form.email" type="email" autocomplete="email" required />
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
          {{ loading ? tr('Проверка...', 'Тексерілуде...') : tr('Продолжить', 'Жалғастыру') }}
        </button>
      </form>

      <form v-else-if="step === 2" class="auth-actions" @submit.prevent="submitVerificationCode">
        <div class="auth-form-grid">
          <label>
            {{ tr('Код подтверждения', 'Растау коды') }}
            <input v-model="codeForm.code" type="text" inputmode="numeric" maxlength="6" required />
          </label>
        </div>

        <p class="auth-inline-note">
          {{ tr('Код отправлен на адрес', 'Код мына адреске жіберілді') }}: <strong>{{ form.email }}</strong>
        </p>

        <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

        <div class="auth-row-actions">
          <button type="button" class="btn btn-ghost" @click="returnToCard">
            {{ tr('Назад', 'Артқа') }}
          </button>
          <button type="submit" class="btn btn-primary" :disabled="loading">
            {{ loading ? tr('Отправка...', 'Жіберілуде...') : tr('Подтвердить и отправить запрос', 'Растап, сұраныс жіберу') }}
          </button>
        </div>
      </form>

      <div v-else class="auth-actions">
        <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>
        <p class="auth-inline-note">
          {{ tr('Администратор увидит запрос в отдельном разделе и примет решение после проверки данных.', 'Әкімші сұранысты бөлек бөлімде көріп, деректерді тексергеннен кейін шешім қабылдайды.') }}
        </p>
        <RouterLink :to="{ name: 'login' }" class="btn btn-primary auth-submit">
          {{ tr('Перейти ко входу', 'Кіру бетіне өту') }}
        </RouterLink>
      </div>

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

.auth-row-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 0.75rem;
}

.auth-inline-note {
  margin: 0;
  color: var(--muted-strong);
}

@media (max-width: 920px) {
  .full-row {
    grid-column: auto;
  }
}
</style>
