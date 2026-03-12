import { computed, ref } from 'vue'

const STORAGE_KEY = 'oper-plan-locale'
const SUPPORTED_LOCALES = ['ru', 'kz']

function detectLocale() {
  if (typeof window === 'undefined') {
    return 'ru'
  }

  const saved = window.localStorage.getItem(STORAGE_KEY)
  if (SUPPORTED_LOCALES.includes(saved)) {
    return saved
  }

  const browser = String(window.navigator.language ?? '').toLowerCase()
  if (browser.startsWith('kk') || browser.startsWith('kz')) {
    return 'kz'
  }

  return 'ru'
}

const locale = ref(detectLocale())

function setLocale(nextLocale) {
  const normalized = String(nextLocale ?? '').toLowerCase()
  if (!SUPPORTED_LOCALES.includes(normalized)) {
    return
  }

  locale.value = normalized

  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, normalized)
  }
}

export function useLocale() {
  const isRu = computed(() => locale.value === 'ru')
  const isKz = computed(() => locale.value === 'kz')

  function tr(ruText, kzText) {
    return isKz.value ? kzText : ruText
  }

  return {
    locale,
    isRu,
    isKz,
    tr,
    setLocale,
    locales: SUPPORTED_LOCALES
  }
}
