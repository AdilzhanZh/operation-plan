<script setup>
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'

const pendingReview = [
  {
    id: 1,
    indicator: 'Increase publication index',
    assignee: 'Vice-Rector for Science',
    status: 'on_review'
  },
  {
    id: 2,
    indicator: 'Launch LMS analytics',
    assignee: 'Vice-Rector for Digitalization',
    status: 'on_review'
  }
]
</script>

<template>
  <section>
    <PageHeader title="Review Queue" subtitle="Tasks awaiting admin approval" />

    <div class="grid">
      <article v-for="item in pendingReview" :key="item.id" class="card">
        <h3>{{ item.indicator }}</h3>
        <p>{{ item.assignee }}</p>
        <StatusBadge :status="item.status" />
      </article>
    </div>
  </section>
</template>

<style scoped>
.grid {
  display: grid;
  gap: 0.9rem;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
}

.card {
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #ffffff;
  padding: 1rem;
}

h3 {
  margin: 0;
  font-size: 1rem;
}

p {
  margin: 0.4rem 0 0.7rem;
  color: #64748b;
}
</style>
