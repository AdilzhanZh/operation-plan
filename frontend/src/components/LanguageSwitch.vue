<script setup>
import { useLocale } from '../composables/useLocale'

const { locale, setLocale } = useLocale()
</script>

<template>
  <div class="lang-switch" role="group" aria-label="Language switch">
    <button
      type="button"
      class="lang-btn"
      :class="{ 'is-active': locale === 'ru' }"
      @click="setLocale('ru')"
    >
      РУС
    </button>
    <button
      type="button"
      class="lang-btn"
      :class="{ 'is-active': locale === 'kz' }"
      @click="setLocale('kz')"
    >
      ҚАЗ
    </button>
  </div>
</template>
