<script setup>
defineProps({
  title: {
    type: String,
    required: true
  },
  subtitle: {
    type: String,
    default: ''
  }
})
</script>

<template>
  <header class="header">
    <h2>{{ title }}</h2>
    <p v-if="subtitle">{{ subtitle }}</p>
  </header>
</template>

<style scoped>
.header {
  margin-bottom: 1rem;
}

h2 {
  margin: 0;
  font-size: 1.45rem;
}

p {
  margin: 0.35rem 0 0;
  color: #64748b;
}
</style>
