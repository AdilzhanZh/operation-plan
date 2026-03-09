<script setup>
import PageHeader from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'

const tasks = [
  {
    id: 1,
    section: 'Academic Excellence',
    indicator: 'Increase publication index',
    deadline: '2026-06-30',
    status: 'in_progress'
  },
  {
    id: 2,
    section: 'Digitalization',
    indicator: 'Launch LMS analytics',
    deadline: '2026-04-15',
    status: 'on_review'
  },
  {
    id: 3,
    section: 'Infrastructure',
    indicator: 'Modernize 3 laboratories',
    deadline: '2026-03-01',
    status: 'overdue'
  }
]
</script>

<template>
  <section>
    <PageHeader title="Tasks" subtitle="Assigned indicators and execution status" />

    <table class="table">
      <thead>
        <tr>
          <th>Section</th>
          <th>Indicator</th>
          <th>Deadline</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="task in tasks" :key="task.id">
          <td>{{ task.section }}</td>
          <td>{{ task.indicator }}</td>
          <td>{{ task.deadline }}</td>
          <td>
            <StatusBadge :status="task.status" />
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<style scoped>
.table {
  width: 100%;
  border-collapse: collapse;
  background: #ffffff;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #e2e8f0;
}

th,
td {
  text-align: left;
  padding: 0.75rem 0.9rem;
  border-bottom: 1px solid #f1f5f9;
}

th {
  background: #f8fafc;
  color: #475569;
  font-weight: 600;
}
</style>
