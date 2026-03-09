<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { loginRequest } from '../services/auth.service'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const authStore = useAuthStore()

const form = reactive({
  username: '',
  password: ''
})

const loading = ref(false)
const errorMessage = ref('')

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
  <section class="login-card">
    <h1>Oper Plan</h1>
    <p>Логин және пароль арқылы кіріңіз</p>

    <form @submit.prevent="submit" class="login-form">
      <label>
        Логин
        <input v-model="form.username" type="text" required />
      </label>

      <label>
        Пароль
        <input v-model="form.password" type="password" required />
      </label>

      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

      <button type="submit" :disabled="loading">
        {{ loading ? 'Кіру...' : 'Кіру' }}
      </button>
    </form>

    <p class="register-link">
      Аккаунтыңыз жоқ па?
      <RouterLink :to="{ name: 'register' }">Тіркелу</RouterLink>
    </p>
  </section>
</template>

<style scoped>
.login-card {
  width: min(440px, 100%);
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
  margin: 0.35rem 0 1.2rem;
  color: #475569;
}

.login-form {
  display: grid;
  gap: 0.9rem;
}

label {
  display: grid;
  gap: 0.4rem;
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

.register-link {
  margin: 1rem 0 0;
}

.register-link a {
  color: #0f766e;
  font-weight: 600;
}
</style>
