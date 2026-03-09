import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

const STORAGE_KEY = 'oper-plan-auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref('')
  const user = ref(null)

  function restore() {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return
    }

    try {
      const parsed = JSON.parse(raw)
      const restoredToken = parsed.token ?? ''
      if (typeof restoredToken !== 'string' || restoredToken.length < 20) {
        token.value = ''
        user.value = null
        localStorage.removeItem(STORAGE_KEY)
        return
      }

      token.value = restoredToken
      user.value = parsed.user ?? null
    } catch {
      token.value = ''
      user.value = null
      localStorage.removeItem(STORAGE_KEY)
    }
  }

  function persist() {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        token: token.value,
        user: user.value
      })
    )
  }

  function login(payload) {
    token.value = payload.token
    user.value = payload.user
    persist()
  }

  function setUser(nextUser) {
    user.value = nextUser
    persist()
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem(STORAGE_KEY)
  }

  const isAuthenticated = computed(() => Boolean(token.value))

  restore()

  return {
    token,
    user,
    isAuthenticated,
    login,
    setUser,
    logout
  }
})
