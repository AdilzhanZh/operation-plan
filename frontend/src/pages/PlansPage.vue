<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import {
  fetchPlanIndicators,
  savePlanIndicator,
  submitPlanIndicatorReport
} from '../services/plan.service'
import { fetchProrectorsRequest } from '../services/user.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()

const selectedYear = ref(String(new Date().getFullYear()))
const rows = ref([])
const prorectors = ref([])
const loading = ref(false)
const savingIndicatorId = ref(null)
const errorMessage = ref('')
const successMessage = ref('')
const selectedResponsibleFilter = ref('')
const assignModalOpen = ref(false)
const activeIndicatorId = ref(null)
const modalSelectedIds = ref([])
const reportModalOpen = ref(false)
const reportIndicatorId = ref(null)
const reportText = ref('')
const reportFiles = ref([])
const reportSending = ref(false)
const nowTimestamp = ref(Date.now())
let countdownIntervalID = null

const isAdmin = computed(() => authStore.user?.role === 'admin')
const isProrector = computed(() => authStore.user?.role === 'prorector')
const canLoadYear = computed(() => selectedYear.value !== '')
const activeReportRow = computed(() => rows.value.find((item) => item.indicator_id === reportIndicatorId.value) ?? null)
const visibleRows = computed(() => {
  if (!isAdmin.value) {
    return rows.value
  }

  const responsibleID = Number(selectedResponsibleFilter.value)
  if (!Number.isInteger(responsibleID) || responsibleID <= 0) {
    return rows.value
  }

  return rows.value.filter((row) => (row.responsible_user_ids ?? []).includes(responsibleID))
})
const hasRows = computed(() => rows.value.length > 0)
const hasVisibleRows = computed(() => visibleRows.value.length > 0)

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

function normalizeIDList(values) {
  if (!Array.isArray(values)) {
    return []
  }

  const unique = new Set()
  for (const value of values) {
    const parsed = Number(value)
    if (Number.isFinite(parsed) && parsed > 0) {
      unique.add(parsed)
    }
  }

  return Array.from(unique)
}

function formatResponsibleNamesByIDs(ids) {
  const normalized = normalizeIDList(ids)
  if (normalized.length === 0) {
    return ''
  }

  return normalized
    .map((id) => prorectors.value.find((item) => Number(item.id) === id))
    .filter(Boolean)
    .map((item) => item.full_name)
    .join(', ')
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

async function loadRows() {
  if (!canLoadYear.value) {
    rows.value = []
    return
  }

  const response = await fetchPlanIndicators(selectedYear.value)
  rows.value = (response.items ?? []).map((item) => ({
    ...item,
    execution_start_date: item.execution_start_date ?? '',
    execution_end_date: item.execution_end_date ?? '',
    schedule_status: item.schedule_status ?? 'not_filled',
    measurement_unit: item.measurement_unit ?? item.unit ?? '',
    responsible_user_ids: Array.isArray(item.responsible_user_ids)
      ? item.responsible_user_ids.map((value) => Number(value))
      : []
  }))

  for (const row of rows.value) {
    if ((!row.responsible || row.responsible.trim() === '') && row.responsible_user_ids.length > 0) {
      row.responsible = formatResponsibleNamesByIDs(row.responsible_user_ids)
    }
  }
}

function formatISODateToDMY(value) {
  const normalized = String(value ?? '').trim()
  if (!normalized) {
    return ''
  }

  const [year, month, day] = normalized.split('-')
  if (!year || !month || !day) {
    return ''
  }
  return `${day}.${month}.${year}`
}

function formatDateRange(row) {
  const start = formatISODateToDMY(row.execution_start_date)
  const end = formatISODateToDMY(row.execution_end_date)
  if (start && end) {
    return `${start} - ${end}`
  }
  const fallback = String(row.execution_deadline ?? '').trim()
  return fallback
}

function parseISODate(value, endOfDay = false) {
  const normalized = String(value ?? '').trim()
  if (!normalized) {
    return null
  }

  const [yearRaw, monthRaw, dayRaw] = normalized.split('-')
  const year = Number(yearRaw)
  const month = Number(monthRaw)
  const day = Number(dayRaw)
  if (!Number.isFinite(year) || !Number.isFinite(month) || !Number.isFinite(day)) {
    return null
  }

  if (endOfDay) {
    return new Date(year, month - 1, day, 23, 59, 59, 999)
  }
  return new Date(year, month - 1, day, 0, 0, 0, 0)
}

function formatRemainingTime(row) {
  if (!row.execution_start_date || !row.execution_end_date) {
    return ''
  }

  const endDate = parseISODate(row.execution_end_date, true)
  if (!endDate) {
    return ''
  }

  const diffMs = endDate.getTime() - nowTimestamp.value
  if (diffMs <= 0) {
    return '00:00:00:00'
  }

  const totalSeconds = Math.floor(diffMs / 1000)
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  return `${String(days).padStart(2, '0')}:${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

function startCountdownTicker() {
  if (typeof window === 'undefined') {
    return
  }
  if (countdownIntervalID !== null) {
    return
  }

  countdownIntervalID = window.setInterval(() => {
    nowTimestamp.value = Date.now()
  }, 1000)
}

function stopCountdownTicker() {
  if (typeof window === 'undefined') {
    return
  }
  if (countdownIntervalID === null) {
    return
  }

  window.clearInterval(countdownIntervalID)
  countdownIntervalID = null
}

function scheduleStatusLabel(status) {
  const normalized = String(status ?? '').toLowerCase()
  if (normalized === 'not_filled') {
    return 'Толық толтырылмаған'
  }
  if (normalized === 'in_progress') {
    return 'Уақыт өтіп жатыр'
  }
  if (normalized === 'overdue') {
    return 'Уақыты өтіп кетті'
  }
  if (normalized === 'upcoming') {
    return 'Уақыты әлі келген жоқ'
  }
  return 'Мерзім қойылмаған'
}

async function loadProrectors() {
  if (!isAdmin.value) {
    prorectors.value = []
    return
  }

  const response = await fetchProrectorsRequest()
  prorectors.value = response.items ?? []
}

async function initialize() {
  loading.value = true
  clearMessages()

  try {
    await loadProrectors()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Plans бөлімін жүктеу кезінде қате болды'
  } finally {
    loading.value = false
  }
}

async function saveRow(row) {
  if (!isAdmin.value || !canLoadYear.value) {
    return
  }

  clearMessages()
  savingIndicatorId.value = row.indicator_id

  try {
    if (!row.execution_start_date || !row.execution_end_date) {
      errorMessage.value = 'Срок исполнения үшін басталу және аяқталу күні міндетті'
      return
    }

    const saved = await savePlanIndicator(row.indicator_id, selectedYear.value, {
      development_indicator: row.development_indicator,
      activities: row.activities,
      execution_start_date: row.execution_start_date,
      execution_end_date: row.execution_end_date,
      responsible_user_ids: normalizeIDList(row.responsible_user_ids)
    })

    Object.assign(row, {
      ...saved,
      execution_start_date: saved?.execution_start_date ?? '',
      execution_end_date: saved?.execution_end_date ?? '',
      schedule_status: saved?.schedule_status ?? 'not_filled',
      responsible_user_ids: Array.isArray(saved?.responsible_user_ids)
        ? saved.responsible_user_ids.map((value) => Number(value))
        : []
    })
    successMessage.value = `Индикатор №${row.indicator_id} сақталды`
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Сақтау кезінде қате болды'
  } finally {
    savingIndicatorId.value = null
  }
}

function openResponsibleModal(row) {
  if (!isAdmin.value) {
    return
  }

  activeIndicatorId.value = row.indicator_id
  modalSelectedIds.value = normalizeIDList(row.responsible_user_ids)
  assignModalOpen.value = true
}

function closeResponsibleModal() {
  assignModalOpen.value = false
  activeIndicatorId.value = null
  modalSelectedIds.value = []
}

async function applyResponsibleSelection() {
  const indicatorID = activeIndicatorId.value
  if (indicatorID === null) {
    closeResponsibleModal()
    return
  }

  const row = rows.value.find((item) => item.indicator_id === indicatorID)
  if (!row) {
    closeResponsibleModal()
    return
  }

  const selectedIDs = normalizeIDList(modalSelectedIds.value)
  row.responsible_user_ids = selectedIDs
  row.responsible = formatResponsibleNamesByIDs(selectedIDs)
  closeResponsibleModal()
  await saveRow(row)
}

function openReportModal(row) {
  if (!isProrector.value) {
    return
  }

  clearMessages()
  reportIndicatorId.value = row.indicator_id
  reportText.value = ''
  reportFiles.value = []
  reportModalOpen.value = true
}

function closeReportModal() {
  reportModalOpen.value = false
  reportIndicatorId.value = null
  reportText.value = ''
  reportFiles.value = []
}

function handleReportFileChange(event) {
  const files = event?.target?.files ?? []
  reportFiles.value = Array.from(files).filter(Boolean)
}

async function submitIndicatorReport() {
  if (!isProrector.value || !canLoadYear.value || reportIndicatorId.value === null) {
    return
  }

  const row = rows.value.find((item) => item.indicator_id === reportIndicatorId.value)
  if (!row) {
    closeReportModal()
    return
  }

  const normalizedText = reportText.value.trim()
  if (reportFiles.value.length === 0) {
    errorMessage.value = 'Кемінде бір файл жүктеу міндетті'
    return
  }

  reportSending.value = true
  clearMessages()

  try {
    await submitPlanIndicatorReport(row.indicator_id, selectedYear.value, {
      report_text: normalizedText,
      files: reportFiles.value
    })
    successMessage.value = `Индикатор №${row.indicator_id} бойынша отчет жіберілді`
    closeReportModal()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Отчет жіберу кезінде қате болды'
  } finally {
    reportSending.value = false
  }
}

onMounted(() => {
  startCountdownTicker()
  initialize()
})

onBeforeUnmount(() => {
  stopCountdownTicker()
})
</script>

<template>
  <section class="plans-page">
    <PageHeader
      title="Plans"
      subtitle="Рабочая матрица текущего года: мероприятия, сроки исполнения и ответственные по каждому индикатору"
      eyebrow="Execution Grid"
    />

    <div class="panel panel-strong toolbar-panel plans-toolbar">
      <div class="plans-year-card">
        <span class="kicker">Current year</span>
        <strong>{{ selectedYear }}</strong>
        <p>Раздел синхронизирован только с индикаторами текущего года.</p>
      </div>

      <label v-if="isAdmin" class="plans-filter">
        <span>Ответственные</span>
        <select v-model="selectedResponsibleFilter">
          <option value="">Барлығы</option>
          <option v-for="prorector in prorectors" :key="`filter-${prorector.id}`" :value="String(prorector.id)">
            {{ prorector.full_name }}
          </option>
        </select>
      </label>

      <div class="plans-visible-card">
        <span class="kicker">Visible</span>
        <strong>{{ visibleRows.length }}</strong>
      </div>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <section class="panel panel-strong plans-table-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">Операционный план</h3>
          <p class="panel-subtitle">Редактирование доступно администратору, отправка отчета - назначенному проректору.</p>
        </div>
        <span class="kicker">{{ visibleRows.length }} indicators</span>
      </div>

      <div v-if="loading" class="empty-state">Загрузка...</div>
      <div v-else-if="!hasRows" class="empty-state">
        {{ selectedYear }} жылына жоспарланған индикатор табылмады.
      </div>
      <div v-else-if="!hasVisibleRows" class="empty-state">
        Таңдалған жауаптыға сәйкес индикатор табылмады.
      </div>

      <div v-else class="table-wrapper">
        <table class="plan-table">
          <thead>
            <tr>
              <th class="col-number">№</th>
              <th>Индикатор Программы развития</th>
              <th>Мероприятия по достижению индикатора</th>
              <th class="col-deadline">Срок исполнения</th>
              <th class="col-responsible">Ответственные</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in visibleRows" :key="row.indicator_id">
              <td class="number-cell">{{ index + 1 }}</td>

              <td>
                <template v-if="isAdmin">
                  <textarea
                    v-model="row.development_indicator"
                    class="plans-textarea indicator-text"
                    rows="4"
                  />
                  <div class="plans-inline-value">
                    {{ formatPlannedValue(row.planned_value, row.measurement_unit || row.unit) }}
                  </div>
                </template>
                <template v-else>
                  <div class="plans-cell-frame text-pretty">{{ row.development_indicator || '—' }}</div>
                  <div class="plans-inline-value">
                    {{ formatPlannedValue(row.planned_value, row.measurement_unit || row.unit) }}
                  </div>
                </template>
              </td>

              <td>
                <template v-if="isAdmin">
                  <textarea v-model="row.activities" class="plans-textarea" rows="5" />
                </template>
                <template v-else>
                  <div class="plans-cell-frame text-pretty">{{ row.activities || '—' }}</div>
                </template>
              </td>

              <td>
                <div class="plans-schedule-card">
                  <template v-if="isAdmin">
                    <div class="date-range-grid">
                      <label class="date-field">
                        <span>Басталуы</span>
                        <input v-model="row.execution_start_date" class="plans-input" type="date" />
                      </label>
                      <label class="date-field">
                        <span>Аяқталуы</span>
                        <input v-model="row.execution_end_date" class="plans-input" type="date" />
                      </label>
                    </div>
                  </template>
                  <template v-else>
                    <div class="plans-cell-frame">{{ formatDateRange(row) || '—' }}</div>
                  </template>

                  <div v-if="isAdmin" class="date-range-preview">
                    {{ formatDateRange(row) || '—' }}
                  </div>
                  <div class="schedule-status" :class="`schedule-${row.schedule_status}`">
                    {{ scheduleStatusLabel(row.schedule_status) }}
                  </div>
                  <div v-if="row.execution_start_date && row.execution_end_date" class="countdown-text">
                    Қалған уақыт: {{ formatRemainingTime(row) }}
                  </div>
                </div>
              </td>

              <td>
                <template v-if="isAdmin">
                  <div class="plans-cell-frame responsible-preview text-pretty">
                    {{ row.responsible || 'Ответственные таңдалмаған' }}
                  </div>
                  <button class="btn btn-ghost plans-assign-btn" type="button" @click="openResponsibleModal(row)">
                    Ответственные бекіту
                  </button>
                  <p v-if="prorectors.length === 0" class="cell-note">
                    Проректорлар тізімі жоқ.
                  </p>
                  <button
                    class="btn btn-primary plans-save-btn"
                    :disabled="savingIndicatorId === row.indicator_id"
                    @click="saveRow(row)"
                  >
                    {{ savingIndicatorId === row.indicator_id ? 'Сақталуда...' : 'Сақтау' }}
                  </button>
                </template>
                <template v-else>
                  <div class="plans-cell-frame text-pretty">{{ row.responsible || '—' }}</div>
                  <button
                    v-if="isProrector"
                    class="btn btn-primary plans-report-btn"
                    type="button"
                    @click="openReportModal(row)"
                  >
                    Отправить отчет
                  </button>
                </template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div v-if="assignModalOpen" class="modal-backdrop" @click.self="closeResponsibleModal">
      <div class="modal-card plans-modal">
        <h3 class="modal-title">Ответственные по индикатору</h3>
        <p class="modal-subtitle">
          Отметьте проректоров, которые будут отвечать за исполнение и отчетность.
        </p>

        <div class="prorector-list">
          <label
            v-for="prorector in prorectors"
            :key="`prorector-option-${prorector.id}`"
            class="prorector-item"
          >
            <input
              v-model="modalSelectedIds"
              type="checkbox"
              :value="Number(prorector.id)"
            />
            <span>
              <strong>{{ prorector.full_name }}</strong>
              <small>{{ prorector.username }}</small>
            </span>
          </label>

          <p v-if="prorectors.length === 0" class="empty-state">
            Проректорлар тізімі бос.
          </p>
        </div>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeResponsibleModal">
            Бас тарту
          </button>
          <button class="btn btn-primary" type="button" @click="applyResponsibleSelection">
            Бекіту
          </button>
        </div>
      </div>
    </div>

    <div v-if="reportModalOpen" class="modal-backdrop" @click.self="closeReportModal">
      <div class="modal-card plans-modal">
        <h3 class="modal-title">Отправить отчет</h3>
        <p class="modal-subtitle">
          {{ activeReportRow?.development_indicator || 'Индикатор' }}
        </p>

        <label class="modal-label">
          Текст отчета
          <textarea
            v-model="reportText"
            class="report-textarea"
            rows="6"
            placeholder="Индикатор бойынша орындалу нәтижесін жазыңыз..."
          />
        </label>

        <label class="modal-label">
          Құжаттар (кемінде 1 файл)
          <input
            class="report-file-input"
            type="file"
            accept=".doc,.docx,.xls,.xlsx,.ppt,.pptx,.pdf"
            multiple
            @change="handleReportFileChange"
          />
        </label>
        <p v-if="reportFiles.length > 0" class="file-list">
          Таңдалған файлдар: {{ reportFiles.map((file) => file.name).join(', ') }}
        </p>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeReportModal">
            Бас тарту
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="reportSending"
            @click="submitIndicatorReport"
          >
            {{ reportSending ? 'Жіберілуде...' : 'Отправить' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.plans-toolbar {
  justify-content: space-between;
}

.plans-year-card,
.plans-visible-card {
  display: grid;
  gap: 0.35rem;
}

.plans-year-card strong,
.plans-visible-card strong {
  font-size: clamp(2rem, 3vw, 2.7rem);
  line-height: 0.95;
  letter-spacing: -0.05em;
}

.plans-year-card p {
  margin: 0;
  color: var(--muted);
  max-width: 19rem;
}

.plans-filter {
  min-width: 18rem;
}

.plans-table-card {
  padding: 1.2rem;
}

.plan-table {
  min-width: 1180px;
}

.col-number {
  width: 4rem;
}

.col-deadline {
  width: 19rem;
}

.col-responsible {
  width: 20rem;
}

.number-cell {
  text-align: center;
  font-weight: 700;
}

.plans-textarea,
.plans-input {
  width: 100%;
}

.indicator-text {
  min-height: 7.5rem;
}

.plans-cell-frame {
  min-height: 6rem;
  padding: 0.9rem;
  border-radius: 18px;
  border: 1px solid rgba(16, 33, 42, 0.08);
  background: rgba(255, 255, 255, 0.68);
}

.plans-inline-value {
  margin-top: 0.6rem;
  color: var(--muted);
  font-size: 0.88rem;
  font-weight: 700;
}

.plans-schedule-card {
  display: grid;
  gap: 0.6rem;
}

.date-range-grid {
  display: grid;
  gap: 0.7rem;
}

.date-field span {
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.date-range-preview {
  color: var(--muted-strong);
  font-size: 0.88rem;
}

.schedule-status {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 0.35rem 0.75rem;
  border-radius: 999px;
  font-size: 0.78rem;
  font-weight: 700;
}

.countdown-text {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--muted-strong);
}

.schedule-in_progress {
  background: rgba(17, 120, 111, 0.12);
  color: #0f5e57;
}

.schedule-overdue {
  background: rgba(183, 75, 58, 0.12);
  color: #a63f32;
}

.schedule-upcoming {
  background: rgba(27, 111, 168, 0.12);
  color: #1b6fa8;
}

.schedule-not_filled {
  background: rgba(201, 111, 59, 0.14);
  color: #9b4b24;
}

.schedule-no_deadline {
  background: rgba(68, 94, 116, 0.1);
  color: #445e74;
}

.responsible-preview {
  min-height: 7rem;
}

.plans-assign-btn,
.plans-save-btn,
.plans-report-btn {
  width: 100%;
  justify-content: center;
  margin-top: 0.65rem;
}

.cell-note {
  margin: 0.6rem 0 0;
  color: var(--muted);
  font-size: 0.82rem;
}

.plans-modal {
  width: min(680px, 100%);
}

.prorector-list {
  display: grid;
  gap: 0.7rem;
  margin-top: 1rem;
}

.prorector-item {
  display: flex;
  gap: 0.8rem;
  align-items: flex-start;
  padding: 0.95rem 1rem;
  border-radius: 20px;
  border: 1px solid rgba(16, 33, 42, 0.08);
  background: rgba(255, 255, 255, 0.72);
}

.prorector-item span {
  display: grid;
  gap: 0.15rem;
}

.prorector-item strong {
  font-size: 0.95rem;
}

.prorector-item small {
  color: var(--muted);
}

.modal-label {
  display: grid;
  gap: 0.45rem;
  margin-top: 1rem;
}

.report-textarea,
.report-file-input {
  width: 100%;
}

.file-list {
  margin: 0.75rem 0 0;
  color: var(--muted);
  font-size: 0.88rem;
}

@media (max-width: 1100px) {
  .plans-toolbar {
    align-items: stretch;
  }
}
</style>
