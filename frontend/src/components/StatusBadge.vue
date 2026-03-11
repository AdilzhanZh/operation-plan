<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: {
    type: String,
    required: true
  }
})

const normalized = computed(() => String(props.status ?? '').toLowerCase())

const labels = {
  created: 'Создан',
  in_progress: 'В работе',
  on_review: 'На проверке',
  completed: 'Завершен',
  overdue: 'Просрочен',
  returned: 'Возвращен'
}
</script>

<template>
  <span class="badge" :class="`badge-${normalized}`">
    <span class="badge-dot" aria-hidden="true"></span>
    {{ labels[normalized] ?? status }}
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.42rem;
  padding: 0.35rem 0.75rem;
  border-radius: 999px;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.01em;
  border: 1px solid transparent;
}

.badge-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 50%;
  background: currentColor;
}

.badge-created {
  background: rgba(68, 94, 116, 0.1);
  border-color: rgba(68, 94, 116, 0.16);
  color: #445e74;
}

.badge-in_progress {
  background: rgba(194, 106, 51, 0.12);
  border-color: rgba(194, 106, 51, 0.18);
  color: #9b4b24;
}

.badge-on_review {
  background: rgba(27, 111, 168, 0.12);
  border-color: rgba(27, 111, 168, 0.18);
  color: #1b6fa8;
}

.badge-completed {
  background: rgba(29, 122, 91, 0.12);
  border-color: rgba(29, 122, 91, 0.18);
  color: #1d7a5b;
}

.badge-overdue {
  background: rgba(183, 75, 58, 0.12);
  border-color: rgba(183, 75, 58, 0.18);
  color: #b74b3a;
}

.badge-returned {
  background: rgba(165, 60, 92, 0.12);
  border-color: rgba(165, 60, 92, 0.18);
  color: #a53c5c;
}
</style>
