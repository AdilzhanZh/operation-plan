<script setup>
import PageHeader from '../components/PageHeader.vue'

const cards = [
  { label: 'Total Tasks', value: 132 },
  { label: 'Completed', value: 87 },
  { label: 'In Progress', value: 29 },
  { label: 'Overdue', value: 16 }
]
</script>

<template>
  <section>
    <PageHeader
      title="Dashboard"
      subtitle="Current execution overview for active operational plan"
    />

    <div class="cards">
      <article v-for="card in cards" :key="card.label" class="card">
        <h3>{{ card.label }}</h3>
        <p>{{ card.value }}</p>
      </article>
    </div>
  </section>
</template>

<style scoped>
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.9rem;
}

.card {
  background: #ffffff;
  border-radius: 10px;
  padding: 1rem;
  border: 1px solid #e2e8f0;
}

h3 {
  margin: 0;
  font-size: 0.92rem;
  color: #64748b;
}

p {
  margin: 0.5rem 0 0;
  font-size: 1.9rem;
  font-weight: 700;
  color: #0f172a;
}
</style>
