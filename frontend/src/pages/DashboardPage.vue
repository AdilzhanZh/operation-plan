<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { fetchPlanIndicators, fetchPlanYears } from '../services/plan.service'

const years = ref([])
const selectedYear = ref('')
const loading = ref(false)
const errorMessage = ref('')
const allRows = ref([])

const activeCard = ref('total')
const stats = ref({
  total: 0,
  completed: 0,
  in_progress: 0,
  overdue: 0
})

const hasYears = computed(() => years.value.length > 0)

const cardConfig = [
  { key: 'total', label: 'Total Tasks' },
  { key: 'completed', label: 'Completed' },
  { key: 'in_progress', label: 'In Progress' },
  { key: 'overdue', label: 'Overdue' }
]

const cards = computed(() => cardConfig.map((card) => ({
  ...card,
  value: stats.value[card.key] ?? 0
})))

const listTitle = computed(() => {
  switch (activeCard.value) {
    case 'completed':
      return 'Completed индикаторлар тізімі'
    case 'in_progress':
      return 'In Progress индикаторлар тізімі'
    case 'overdue':
      return 'Overdue индикаторлар тізімі'
    default:
      return 'Барлық индикаторлар тізімі'
  }
})

function statusLabel(status) {
  const normalized = String(status ?? '').toLowerCase()
  if (normalized === 'completed') {
    return 'Completed'
  }
  if (normalized === 'overdue') {
    return 'Overdue'
  }
  return 'In Progress'
}

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

function statusFilterByCard(cardKey) {
  switch (cardKey) {
    case 'completed':
      return 'completed'
    case 'in_progress':
      return 'in_progress'
    case 'overdue':
      return 'overdue'
    default:
      return ''
  }
}

function deriveDashboardStatus(row) {
  const reportStatus = String(row?.report_status ?? '').toLowerCase()
  if (reportStatus === 'completed') {
    return 'completed'
  }

  const scheduleStatus = String(row?.schedule_status ?? '').toLowerCase()
  if (scheduleStatus === 'overdue') {
    return 'overdue'
  }

  return 'in_progress'
}

function recalculateStats() {
  const next = {
    total: allRows.value.length,
    completed: 0,
    in_progress: 0,
    overdue: 0
  }

  for (const row of allRows.value) {
    const status = String(row.dashboard_status ?? '').toLowerCase()
    if (status === 'completed') {
      next.completed += 1
      continue
    }
    if (status === 'overdue') {
      next.overdue += 1
      continue
    }
    next.in_progress += 1
  }

  stats.value = next
}

const rows = computed(() => {
  const filter = statusFilterByCard(activeCard.value)
  if (!filter) {
    return allRows.value
  }

  return allRows.value.filter((row) => String(row.dashboard_status).toLowerCase() === filter)
})

async function loadYears() {
  const response = await fetchPlanYears()
  const currentYear = new Date().getFullYear()
  const sourceYears = Array.isArray(response.years) ? response.years : []
  const currentYearRows = sourceYears.filter((year) => Number(year) === currentYear)

  years.value = currentYearRows.length > 0 ? currentYearRows : [currentYear]
  selectedYear.value = String(currentYear)
}

async function loadRows() {
  if (!selectedYear.value) {
    allRows.value = []
    stats.value = {
      total: 0,
      completed: 0,
      in_progress: 0,
      overdue: 0
    }
    return
  }

  const response = await fetchPlanIndicators(selectedYear.value)
  allRows.value = (response.items ?? []).map((item) => ({
    ...item,
    dashboard_status: deriveDashboardStatus(item)
  }))
  recalculateStats()
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
      ?? 'Dashboard жүктеу кезінде қате болды'
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
      ?? 'Жыл бойынша деректі жүктеу мүмкін болмады'
  } finally {
    loading.value = false
  }
}

async function handleCardClick(cardKey) {
  if (activeCard.value === cardKey) {
    return
  }

  activeCard.value = cardKey
}

onMounted(() => {
  initialize()
})
</script>

<template>
  <section class="dashboard-page">
    <PageHeader
      title="Dashboard"
      subtitle="Current execution overview for active operational plan"
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

    <div class="cards">
      <button
        v-for="card in cards"
        :key="card.key"
        class="card-button"
        :class="{ 'is-active': activeCard === card.key }"
        type="button"
        @click="handleCardClick(card.key)"
      >
        <h3>{{ card.label }}</h3>
        <p>{{ card.value }}</p>
      </button>
    </div>

    <section class="list-section">
      <h2>{{ listTitle }}</h2>

      <div v-if="loading" class="state-box">Загрузка...</div>
      <div v-else-if="!hasYears" class="state-box">Әзірге жылдық дерек жоқ</div>
      <div v-else-if="rows.length === 0" class="state-box">Тізім бос</div>
      <div v-else class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>№</th>
              <th>Целевой индикатор</th>
              <th>Мән</th>
              <th>Срок исполнения</th>
              <th>Ответственные</th>
              <th>Статус</th>
              <th>Отправлено</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.indicator_id">
              <td class="number-cell">{{ index + 1 }}</td>
              <td>{{ row.development_indicator || '—' }}</td>
              <td>{{ formatPlannedValue(row.planned_value, row.unit) }}</td>
              <td>{{ row.execution_deadline || '—' }}</td>
              <td>{{ row.responsible || '—' }}</td>
              <td>
                <span class="status-pill" :class="`status-${row.dashboard_status}`">
                  {{ statusLabel(row.dashboard_status) }}
                </span>
              </td>
              <td>{{ formatDate(row.last_submitted_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>

<style scoped>
.dashboard-page {
  display: grid;
  gap: 0.9rem;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
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

.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 0.9rem;
}

.card-button {
  text-align: left;
  background: #ffffff;
  border-radius: 10px;
  padding: 1rem;
  border: 1px solid #e2e8f0;
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.card-button.is-active {
  border-color: #0f766e;
  box-shadow: 0 0 0 2px rgba(15, 118, 110, 0.15);
}

.card-button h3 {
  margin: 0;
  font-size: 0.92rem;
  color: #64748b;
}

.card-button p {
  margin: 0.5rem 0 0;
  font-size: 1.9rem;
  font-weight: 700;
  color: #0f172a;
}

.list-section {
  display: grid;
  gap: 0.55rem;
}

.list-section h2 {
  margin: 0;
  font-size: 1.08rem;
}

.table-wrap {
  overflow-x: auto;
  border: 1px solid #d8e0ea;
  border-radius: 10px;
  background: #ffffff;
}

.table {
  width: 100%;
  min-width: 940px;
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
  font-weight: 700;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 92px;
  border-radius: 999px;
  padding: 0.3rem 0.65rem;
  font-size: 0.82rem;
  font-weight: 700;
}

.status-pending {
  background: #fef3c7;
  color: #92400e;
}

.status-completed {
  background: #dcfce7;
  color: #166534;
}

.status-in_progress {
  background: #fef3c7;
  color: #92400e;
}

.status-overdue {
  background: #fee2e2;
  color: #991b1b;
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
