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

    <div class="page-header-metrics" aria-hidden="true">
      <span class="page-header-metric page-header-metric-main"></span>
      <span class="page-header-metric page-header-metric-accent"></span>
      <span class="page-header-metric page-header-metric-soft"></span>
    </div>
  </header>
</template>

<style scoped>
.page-header {
  position: relative;
  overflow: hidden;
  display: flex;
  justify-content: space-between;
  gap: 1.25rem;
  align-items: flex-start;
  padding: 1.5rem 1.65rem;
  border-radius: 30px;
  border: 1px solid rgba(203, 209, 234, 0.7);
  background:
    radial-gradient(circle at top left, rgba(160, 235, 225, 0.34), transparent 32%),
    radial-gradient(circle at right 12% top 10%, rgba(126, 122, 243, 0.18), transparent 22%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(247, 248, 255, 0.96));
  box-shadow: 0 20px 56px rgba(108, 117, 165, 0.12);
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
  margin-bottom: 0.85rem;
  padding: 0.45rem 0.78rem;
  border-radius: 999px;
  background: rgba(126, 122, 243, 0.1);
  color: #6663d7;
  font-size: 0.76rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

h2 {
  margin: 0;
  font-size: clamp(2rem, 3vw, 2.9rem);
  line-height: 0.98;
  letter-spacing: -0.04em;
  color: #1c2440;
}

p {
  margin: 0.9rem 0 0;
  max-width: 42rem;
  color: #737a96;
  font-size: 1rem;
}

.page-header-metrics {
  position: relative;
  flex: 0 0 8.75rem;
  min-height: 7.5rem;
}

.page-header-metric {
  position: absolute;
  display: block;
  border-radius: 26px;
  border: 1px solid rgba(204, 209, 236, 0.8);
  backdrop-filter: blur(18px);
}

.page-header-metric-main {
  inset: 0.4rem 0.1rem auto auto;
  width: 5.8rem;
  height: 5.8rem;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.74), rgba(255, 255, 255, 0.28)),
    linear-gradient(140deg, rgba(126, 122, 243, 0.34), rgba(163, 235, 226, 0.28));
  box-shadow: 0 18px 44px rgba(111, 120, 179, 0.16);
}

.page-header-metric-accent {
  inset: auto 2.4rem 0.4rem auto;
  width: 2.8rem;
  height: 2.8rem;
  border-radius: 20px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.74), rgba(255, 255, 255, 0.24)),
    linear-gradient(145deg, rgba(163, 235, 226, 0.72), rgba(126, 122, 243, 0.24));
}

.page-header-metric-soft {
  inset: 1rem auto auto 1rem;
  width: 3.25rem;
  height: 3.25rem;
  border-radius: 22px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.7), rgba(255, 255, 255, 0.3)),
    linear-gradient(145deg, rgba(255, 222, 196, 0.58), rgba(255, 255, 255, 0.24));
}

@media (max-width: 720px) {
  .page-header {
    padding: 1.25rem;
    border-radius: 24px;
  }

  .page-header-metrics {
    display: none;
  }

  h2 {
    font-size: 1.85rem;
  }
}
</style>
