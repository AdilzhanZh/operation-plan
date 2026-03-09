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
  <section class="register-card">
    <h1>Тіркелу</h1>
    <p>Жаңа пайдаланушы (әдепкі рөл: viewer)</p>

    <form @submit.prevent="submit" class="register-form">
      <label>
        Аты
        <input v-model="form.first_name" type="text" required />
      </label>

      <label>
        Фамилиясы
        <input v-model="form.last_name" type="text" required />
      </label>

      <label>
        Тегі
        <input v-model="form.middle_name" type="text" />
      </label>

      <label>
        Логин
        <input v-model="form.username" type="text" required />
      </label>

      <label>
        Пароль
        <input v-model="form.password" type="password" required />
      </label>

      <label>
        Пароль (қайта)
        <input v-model="form.confirm_password" type="password" required />
      </label>

      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
      <p v-if="successMessage" class="success">{{ successMessage }}</p>

      <button type="submit" :disabled="loading">
        {{ loading ? 'Тіркелу...' : 'Тіркелу' }}
      </button>
    </form>

    <p class="login-link">
      Аккаунт бар ма?
      <RouterLink :to="{ name: 'login' }">Кіру</RouterLink>
    </p>
  </section>
</template>

<style scoped>
.register-card {
  width: min(520px, 100%);
  margin: 0 auto;
  background: #ffffff;
  border-radius: 12px;
  padding: 2rem;
  box-shadow: 0 10px 25px rgb(15 23 42 / 0.08);
}

h1 {
  margin: 0;
  font-size: 1.6rem;
}

p {
  margin: 0.35rem 0 1rem;
  color: #475569;
}

.register-form {
  display: grid;
  gap: 0.8rem;
}

label {
  display: grid;
  gap: 0.35rem;
  font-size: 0.92rem;
}

input,
button {
  width: 100%;
  border-radius: 8px;
  border: 1px solid #cbd5e1;
  padding: 0.6rem 0.8rem;
  font: inherit;
}

button {
  border: 0;
  background: #0f172a;
  color: #f8fafc;
  font-weight: 600;
  cursor: pointer;
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.error {
  margin: 0;
  color: #991b1b;
}

.success {
  margin: 0;
  color: #166534;
}

.login-link {
  margin-top: 1rem;
}

.login-link a {
  color: #0f766e;
  font-weight: 600;
}
</style>
