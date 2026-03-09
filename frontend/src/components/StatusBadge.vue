<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: {
    type: String,
    required: true
  }
})

const normalized = computed(() => props.status.toLowerCase())

const labels = {
  created: 'Created',
  in_progress: 'In Progress',
  on_review: 'On Review',
  completed: 'Completed',
  overdue: 'Overdue',
  returned: 'Returned'
}
</script>

<template>
  <span class="badge" :class="`badge-${normalized}`">
    {{ labels[normalized] ?? status }}
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.55rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge-created {
  background: #e5e7eb;
  color: #374151;
}

.badge-in_progress {
  background: #fef3c7;
  color: #92400e;
}

.badge-on_review {
  background: #dbeafe;
  color: #1d4ed8;
}

.badge-completed {
  background: #dcfce7;
  color: #166534;
}

.badge-overdue {
  background: #fee2e2;
  color: #991b1b;
}

.badge-returned {
  background: #ffe4e6;
  color: #9f1239;
}
</style>
