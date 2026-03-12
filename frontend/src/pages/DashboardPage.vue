<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { fetchPlanIndicators, fetchPlanYears } from '../services/plan.service'
import { useAuthStore } from '../store/auth'

const years = ref([])
const selectedYear = ref('')
const loading = ref(false)
const errorMessage = ref('')
const allRows = ref([])
const authStore = useAuthStore()
const isProrector = computed(() => authStore.user?.role === 'prorector')
const readModalOpen = ref(false)
const readModalTitle = ref('')
const readModalText = ref('')

const activeCard = ref('total')
const stats = ref({
  total: 0,
  completed: 0,
  not_filled: 0,
  in_progress: 0,
  overdue: 0
})

const hasYears = computed(() => years.value.length > 0)

const cardMeta = {
  total: 'Полный пул индикаторов выбранного года',
  completed: 'Позиции с утвержденным завершением',
  not_filled: 'Индикаторы без заполненного графика или отчета',
  in_progress: 'Активные задачи в пределах срока',
  overdue: 'Точки, требующие немедленного контроля'
}

const cardConfig = computed(() => {
  if (isProrector.value) {
    return [
      { key: 'total', label: 'Total Tasks' },
      { key: 'completed', label: 'Completed' },
      { key: 'overdue', label: 'Overdue' }
    ]
  }
  return [
    { key: 'total', label: 'Total Tasks' },
    { key: 'completed', label: 'Completed' },
    { key: 'not_filled', label: 'Not Filled' },
    { key: 'in_progress', label: 'In Progress' },
    { key: 'overdue', label: 'Overdue' }
  ]
})

const cards = computed(() => cardConfig.value.map((card) => ({
  ...card,
  value: stats.value[card.key] ?? 0,
  meta: cardMeta[card.key] ?? ''
})))

const listTitle = computed(() => {
  switch (activeCard.value) {
    case 'completed':
      return 'Completed индикаторлар тізімі'
    case 'in_progress':
      return 'In Progress индикаторлар тізімі'
    case 'not_filled':
      return 'Not Filled индикаторлар тізімі'
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
  if (normalized === 'not_filled') {
    return 'Not Filled'
  }
  return 'In Progress'
}

function textPreview(value) {
  const normalized = String(value ?? '').trim()
  return normalized || '—'
}

function openReadModal(title, value) {
  readModalTitle.value = title
  readModalText.value = textPreview(value)
  readModalOpen.value = true
}

function closeReadModal() {
  readModalOpen.value = false
  readModalTitle.value = ''
  readModalText.value = ''
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
  const raw = String(value ?? '').trim()
  if (!raw) {
    return '—'
  }

  let normalized = raw
  normalized = normalized.replace(/^(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2}:\d{2})/, '$1T$2')
  normalized = normalized.replace(/([+-]\d{2})(\d{2})$/, '$1:$2')
  normalized = normalized.replace(/([+-]\d{2})$/, '$1:00')

  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) {
    return raw
  }
  return date.toLocaleString('ru-RU')
}

function statusFilterByCard(cardKey) {
  switch (cardKey) {
    case 'completed':
      return 'completed'
    case 'in_progress':
      return 'in_progress'
    case 'not_filled':
      return 'not_filled'
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
  if (scheduleStatus === 'not_filled' || scheduleStatus === 'no_deadline') {
    return isProrector.value ? 'in_progress' : 'not_filled'
  }
  if (scheduleStatus === 'overdue') {
    return 'overdue'
  }

  return 'in_progress'
}

function recalculateStats() {
  const next = {
    total: allRows.value.length,
    completed: 0,
    not_filled: 0,
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
    if (status === 'not_filled') {
      next.not_filled += 1
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
  const normalizedYears = Array.isArray(response.years)
    ? response.years
        .map((year) => Number(year))
        .filter((year) => Number.isInteger(year))
        .sort((a, b) => a - b)
    : []

  years.value = normalizedYears

  if (normalizedYears.length === 0) {
    selectedYear.value = ''
    return
  }

  const currentSelection = Number(selectedYear.value)
  if (Number.isInteger(currentSelection) && normalizedYears.includes(currentSelection)) {
    return
  }

  const preferredYear = normalizedYears.includes(currentYear)
    ? currentYear
    : normalizedYears[normalizedYears.length - 1]

  selectedYear.value = String(preferredYear)
}

async function loadRows() {
  if (!selectedYear.value) {
    allRows.value = []
    stats.value = {
      total: 0,
      completed: 0,
      not_filled: 0,
      in_progress: 0,
      overdue: 0
    }
    return
  }

  const response = await fetchPlanIndicators(selectedYear.value, {
    include_submitted: isProrector.value
  })
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

function handleCardClick(cardKey) {
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
      subtitle="Ключевой обзор по индикаторам, срокам исполнения и последним отправленным отчетам за активный год"
      eyebrow="Overview"
    />

    <div class="panel panel-strong toolbar-panel dashboard-toolbar">
      <label class="dashboard-filter">
        <span>Год</span>
        <select :value="selectedYear" :disabled="loading || !hasYears" @change="handleYearChange">
          <option v-if="!hasYears" value="">Нет годов</option>
          <option v-for="year in years" :key="year" :value="String(year)">
            {{ year }}
          </option>
        </select>
      </label>

      <div class="dashboard-toolbar-note">
        <span class="kicker">Focus</span>
        <p>Выберите карточку ниже, чтобы быстро сфокусировать список по нужному статусу.</p>
      </div>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>

    <div class="dashboard-stats">
      <button
        v-for="card in cards"
        :key="card.key"
        type="button"
        class="unstyled-button dashboard-stat-card"
        :class="{ 'is-active': activeCard === card.key }"
        @click="handleCardClick(card.key)"
      >
        <span class="dashboard-stat-label">{{ card.label }}</span>
        <strong>{{ card.value }}</strong>
        <p>{{ card.meta }}</p>
      </button>
    </div>

    <section class="panel panel-strong dashboard-list-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ listTitle }}</h3>
          <p class="panel-subtitle">Список автоматически перестраивается по выбранной карточке статуса.</p>
        </div>
        <span class="kicker">{{ rows.length }} rows</span>
      </div>

      <div v-if="loading" class="empty-state">Загрузка...</div>
      <div v-else-if="!hasYears" class="empty-state">Әзірге жылдық дерек жоқ</div>
      <div v-else-if="rows.length === 0" class="empty-state">Тізім бос</div>
      <div v-else class="table-wrap">
        <table class="table dashboard-table">
          <thead>
            <tr>
              <th>№</th>
              <th>Целевой индикатор</th>
              <th>Срок исполнения</th>
              <th>Ответственные</th>
              <th>Статус</th>
              <th>Отправлено</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.indicator_id">
              <td class="number-cell" data-label="№">{{ index + 1 }}</td>
              <td data-label="Целевой индикатор">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.development_indicator) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal('Целевой индикатор', row.development_indicator)"
                  @keyup.enter="openReadModal('Целевой индикатор', row.development_indicator)"
                  @keyup.space.prevent="openReadModal('Целевой индикатор', row.development_indicator)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.development_indicator) }}</span>
                </div>
                <span class="planned-value-chip">{{ formatPlannedValue(row.planned_value, row.unit) }}</span>
              </td>
              <td data-label="Срок исполнения">{{ row.execution_deadline || '—' }}</td>
              <td class="text-pretty" data-label="Ответственные">{{ row.responsible || '—' }}</td>
              <td data-label="Статус">
                <span class="status-pill" :class="`status-${row.dashboard_status}`">
                  {{ statusLabel(row.dashboard_status) }}
                </span>
              </td>
              <td data-label="Отправлено">{{ formatDate(row.last_submitted_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div v-if="readModalOpen" class="modal-backdrop" @click.self="closeReadModal">
      <div class="modal-card dashboard-read-modal">
        <h3 class="modal-title">{{ readModalTitle }}</h3>
        <div class="dashboard-read-content text-pretty">
          {{ readModalText }}
        </div>
        <div class="modal-actions">
          <button class="btn btn-primary" type="button" @click="closeReadModal">Жабу</button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.dashboard-toolbar {
  justify-content: space-between;
}

.dashboard-filter {
  min-width: 13rem;
}

.dashboard-toolbar-note {
  display: grid;
  gap: 0.45rem;
  max-width: 26rem;
}

.dashboard-toolbar-note p {
  margin: 0;
  color: var(--muted);
}

.dashboard-stats {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
}

.dashboard-stat-card {
  display: grid;
  gap: 0.6rem;
  padding: 1.25rem;
  border-radius: 24px;
  border: 1px solid rgba(16, 33, 42, 0.1);
  background:
    radial-gradient(circle at top right, rgba(17, 120, 111, 0.1), transparent 36%),
    linear-gradient(145deg, rgba(255, 251, 245, 0.96), rgba(255, 255, 255, 0.92));
  box-shadow: var(--shadow-soft);
  text-align: left;
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.dashboard-stat-card:hover {
  transform: translateY(-2px);
}

.dashboard-stat-card.is-active {
  border-color: rgba(17, 120, 111, 0.32);
  box-shadow: 0 22px 40px rgba(17, 120, 111, 0.14);
}

.dashboard-stat-label {
  color: var(--muted);
  font-size: 0.8rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.dashboard-stat-card strong {
  font-size: clamp(2rem, 4vw, 3rem);
  line-height: 0.95;
  letter-spacing: -0.05em;
}

.dashboard-stat-card p {
  margin: 0;
  color: var(--muted);
  font-size: 0.88rem;
}

.dashboard-list-card {
  padding: 1.2rem;
}

.dashboard-table {
  min-width: 900px;
}

.number-cell {
  width: 4rem;
  text-align: center;
  font-weight: 700;
}

.planned-value-chip {
  display: inline-flex;
  align-items: center;
  margin-top: 0.4rem;
  border: 1px solid #d8e0ea;
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  font-size: 0.78rem;
  font-weight: 700;
  color: #475569;
  background: #f8fafc;
}

.dashboard-read-modal {
  width: min(760px, 100%);
}

.dashboard-read-content {
  max-height: min(60vh, 460px);
  overflow: auto;
  margin-top: 0.6rem;
  padding: 0.9rem 1rem;
  border-radius: 14px;
  border: 1px solid rgba(16, 33, 42, 0.1);
  background: rgba(255, 255, 255, 0.72);
  white-space: pre-wrap;
  line-height: 1.5;
}

@media (max-width: 900px) {
  .dashboard-toolbar {
    align-items: stretch;
  }
}

@media (max-width: 840px) {
  .dashboard-list-card {
    padding: 0.95rem;
  }

  .table-wrap {
    overflow: visible;
    border: 0;
    box-shadow: none;
    background: transparent;
  }

  .dashboard-table {
    min-width: 0;
    display: block;
  }

  .dashboard-table thead {
    display: none;
  }

  .dashboard-table tbody {
    display: grid;
    gap: 0.78rem;
  }

  .dashboard-table tbody tr {
    display: block;
    padding: 0.72rem 0.82rem;
    border: 1px solid var(--border);
    border-radius: 18px;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: var(--shadow-soft);
  }

  .dashboard-table tbody td {
    display: grid;
    grid-template-columns: minmax(120px, 38%) 1fr;
    gap: 0.52rem;
    padding: 0.48rem 0.1rem;
    border-bottom: 1px dashed rgba(16, 33, 42, 0.12);
  }

  .dashboard-table tbody td:last-child {
    border-bottom: 0;
  }

  .dashboard-table tbody td::before {
    content: attr(data-label);
    color: var(--muted);
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .number-cell {
    text-align: left;
  }
}

@media (max-width: 620px) {
  .dashboard-stats {
    grid-template-columns: 1fr;
  }

  .dashboard-table tbody td {
    grid-template-columns: 1fr;
    gap: 0.38rem;
  }

  .dashboard-table tbody td::before {
    font-size: 0.68rem;
  }
}
</style>
