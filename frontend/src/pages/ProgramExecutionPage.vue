<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { fetchPlanReports, fetchPlanYears } from '../services/plan.service'

const years = ref([])
const selectedYear = ref('')
const rows = ref([])
const loading = ref(false)
const errorMessage = ref('')

const hasYears = computed(() => years.value.length > 0)
const canLoadYear = computed(() => selectedYear.value !== '')

function formatPlannedValue(value, unit) {
  const rawValue = String(value ?? '').trim()
  const rawUnit = String(unit ?? '').trim()

  if (rawValue === '') {
    return '—'
  }
  if (rawUnit === '') {
    return rawValue
  }
  if (rawUnit === '%') {
    return `${rawValue}%`
  }
  return `${rawValue} ${rawUnit}`
}

function formatDate(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '—'
  }
  return date.toLocaleString('ru-RU')
}

async function loadYears() {
  const response = await fetchPlanYears()
  years.value = response.years ?? []
  if (!selectedYear.value && years.value.length > 0) {
    selectedYear.value = String(years.value[years.value.length - 1])
  }
}

async function loadRows() {
  if (!canLoadYear.value) {
    rows.value = []
    return
  }

  const response = await fetchPlanReports(selectedYear.value)
  rows.value = response.items ?? []
}

async function initialize() {
  loading.value = true
  errorMessage.value = ''

  try {
    await loadYears()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Мәліметтерді жүктеу мүмкін болмады'
  } finally {
    loading.value = false
  }
}

async function handleYearChange(event) {
  selectedYear.value = event.target.value
  loading.value = true
  errorMessage.value = ''

  try {
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Жыл бойынша мәліметтерді жүктеу мүмкін болмады'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  initialize()
})
</script>

<template>
  <section class="execution-page">
    <PageHeader
      title="ВЫПОЛНЕНИЕ ПРОГРАММЫ РАЗВИТИЯ"
      subtitle="Проректорлар жіберген индикаторлар бойынша есептер"
    />

    <div class="toolbar">
      <label class="year-picker">
        <span>Год:</span>
        <select :value="selectedYear" :disabled="loading || !hasYears" @change="handleYearChange">
          <option v-if="!hasYears" value="">Нет годов</option>
          <option v-for="year in years" :key="year" :value="String(year)">
            {{ year }}
          </option>
        </select>
      </label>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>

    <div v-if="loading" class="state-box">Загрузка...</div>
    <div v-else-if="!hasYears" class="state-box">
      Әзірге жылдар жоқ.
    </div>
    <div v-else-if="rows.length === 0" class="state-box">
      {{ selectedYear }} жылына жіберілген отчеттар жоқ.
    </div>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>№</th>
            <th>Целевой индикатор</th>
            <th>Мән (ед. изм.)</th>
            <th>Срок исполнения</th>
            <th>Ответственные</th>
            <th>Выполнение индикатора</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in rows" :key="row.id">
            <td class="number-cell">{{ index + 1 }}</td>
            <td>{{ row.development_indicator }}</td>
            <td>{{ formatPlannedValue(row.planned_value, row.unit) }}</td>
            <td>{{ row.execution_deadline || '—' }}</td>
            <td>{{ row.responsible || '—' }}</td>
            <td>
              <p class="report-text">{{ row.report_text || '—' }}</p>
              <a v-if="row.file_path" :href="row.file_path" target="_blank" rel="noopener noreferrer">
                {{ row.file_name || row.file_path }}
              </a>
              <p class="meta">
                Отправил: {{ row.submitted_by_name || row.submitted_by }} • {{ formatDate(row.submitted_at) }}
              </p>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.execution-page {
  display: grid;
  gap: 0.9rem;
}

.toolbar {
  display: flex;
  justify-content: flex-start;
}

.year-picker {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  font-size: 0.92rem;
}

.year-picker select {
  min-width: 140px;
  padding: 0.45rem 0.6rem;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  background: #ffffff;
}

.table-wrap {
  overflow-x: auto;
  border: 1px solid #d8e0ea;
  border-radius: 10px;
  background: #ffffff;
}

.table {
  width: 100%;
  min-width: 1100px;
  border-collapse: collapse;
}

.table th,
.table td {
  border: 1px solid #d8e0ea;
  padding: 0.6rem;
  vertical-align: top;
}

.table thead th {
  background: #f4f6f8;
  text-align: center;
  font-weight: 700;
}

.number-cell {
  text-align: center;
  font-weight: 600;
}

.report-text {
  margin: 0 0 0.4rem;
  white-space: pre-wrap;
}

.meta {
  margin: 0.45rem 0 0;
  font-size: 0.8rem;
  color: #64748b;
}

.message {
  margin: 0;
  padding: 0.65rem 0.85rem;
  border-radius: 8px;
  font-size: 0.92rem;
}

.message-error {
  background: #fee2e2;
  color: #991b1b;
}

.state-box {
  border: 1px dashed #cbd5e1;
  border-radius: 10px;
  background: #f8fafc;
  padding: 1rem;
  color: #475569;
}
</style>
