<script setup>
defineProps({
  title: {
    type: String,
    required: true
  },
  subtitle: {
    type: String,
    default: ''
  },
  eyebrow: {
    type: String,
    default: 'Операционное планирование'
  }
})
</script>

<template>
  <header class="page-header">
    <div class="page-header-copy">
      <span class="page-header-eyebrow">{{ eyebrow }}</span>
      <h2>{{ title }}</h2>
      <p v-if="subtitle">{{ subtitle }}</p>
    </div>

    <div class="page-header-orbit" aria-hidden="true">
      <span></span>
      <span></span>
      <span></span>
    </div>
  </header>
</template>

<style scoped>
.page-header {
  position: relative;
  overflow: hidden;
  display: flex;
  justify-content: space-between;
  gap: 1.5rem;
  align-items: flex-start;
  padding: 1.5rem 1.6rem;
  border-radius: 28px;
  border: 1px solid rgba(10, 38, 48, 0.1);
  background:
    radial-gradient(circle at top left, rgba(18, 132, 125, 0.18), transparent 36%),
    linear-gradient(140deg, rgba(255, 251, 245, 0.98), rgba(255, 255, 255, 0.94));
  box-shadow: 0 26px 80px rgba(15, 31, 41, 0.1);
}

.page-header-copy {
  position: relative;
  z-index: 1;
  max-width: 48rem;
}

.page-header-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  margin-bottom: 0.75rem;
  padding: 0.38rem 0.72rem;
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.1);
  color: #0e6c67;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

h2 {
  margin: 0;
  font-size: clamp(1.9rem, 3vw, 2.75rem);
  line-height: 1.02;
  letter-spacing: -0.04em;
  color: #10212a;
}

p {
  margin: 0.75rem 0 0;
  max-width: 42rem;
  color: #50616a;
  font-size: 1rem;
}

.page-header-orbit {
  position: relative;
  flex: 0 0 10rem;
  min-height: 7rem;
}

.page-header-orbit span {
  position: absolute;
  display: block;
  border-radius: 999px;
  border: 1px solid rgba(10, 38, 48, 0.08);
  background: rgba(255, 255, 255, 0.56);
  backdrop-filter: blur(12px);
}

.page-header-orbit span:nth-child(1) {
  inset: 0.4rem 0.2rem auto auto;
  width: 5.6rem;
  height: 5.6rem;
  background: linear-gradient(145deg, rgba(201, 111, 59, 0.28), rgba(255, 255, 255, 0.18));
}

.page-header-orbit span:nth-child(2) {
  inset: auto 2.2rem 0.2rem auto;
  width: 2.5rem;
  height: 2.5rem;
  background: linear-gradient(145deg, rgba(15, 118, 110, 0.3), rgba(255, 255, 255, 0.18));
}

.page-header-orbit span:nth-child(3) {
  inset: 1rem auto auto 1.2rem;
  width: 3.1rem;
  height: 3.1rem;
}

@media (max-width: 720px) {
  .page-header {
    padding: 1.25rem;
    border-radius: 24px;
  }

  .page-header-orbit {
    display: none;
  }

  h2 {
    font-size: 1.85rem;
  }
}
</style>
