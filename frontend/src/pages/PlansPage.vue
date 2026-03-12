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
const reportModalOpen = ref(false)
const reportIndicatorId = ref(null)
const reportText = ref('')
const reportFiles = ref([])
const reportSending = ref(false)
const readModalOpen = ref(false)
const readModalTitle = ref('')
const readModalText = ref('')
const rowEditModalOpen = ref(false)
const rowEditIndicatorId = ref(null)
const rowEditForm = ref({
  development_indicator: '',
  activities: '',
  execution_start_date: '',
  execution_end_date: '',
  responsible_user_ids: []
})
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
const activeRowEdit = computed(() => rows.value.find((item) => item.indicator_id === rowEditIndicatorId.value) ?? null)

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
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

function openRowEditModal(row) {
  if (!isAdmin.value) {
    return
  }

  rowEditIndicatorId.value = row.indicator_id
  rowEditForm.value = {
    development_indicator: String(row.development_indicator ?? ''),
    activities: String(row.activities ?? ''),
    execution_start_date: String(row.execution_start_date ?? ''),
    execution_end_date: String(row.execution_end_date ?? ''),
    responsible_user_ids: normalizeIDList(row.responsible_user_ids)
  }
  rowEditModalOpen.value = true
  clearMessages()
}

function closeRowEditModal() {
  rowEditModalOpen.value = false
  rowEditIndicatorId.value = null
  rowEditForm.value = {
    development_indicator: '',
    activities: '',
    execution_start_date: '',
    execution_end_date: '',
    responsible_user_ids: []
  }
}

async function saveRowFromModal() {
  if (!isAdmin.value || !canLoadYear.value || rowEditIndicatorId.value === null) {
    return
  }

  const row = rows.value.find((item) => item.indicator_id === rowEditIndicatorId.value)
  if (!row) {
    closeRowEditModal()
    return
  }

  row.development_indicator = rowEditForm.value.development_indicator.trim()
  row.activities = rowEditForm.value.activities.trim()
  row.execution_start_date = String(rowEditForm.value.execution_start_date ?? '')
  row.execution_end_date = String(rowEditForm.value.execution_end_date ?? '')
  row.responsible_user_ids = normalizeIDList(rowEditForm.value.responsible_user_ids)
  row.responsible = formatResponsibleNamesByIDs(row.responsible_user_ids)

  await saveRow(row)
  if (!errorMessage.value) {
    closeRowEditModal()
  }
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
              <td class="number-cell" data-label="№">{{ index + 1 }}</td>

              <td data-label="Индикатор программы развития">
                <div
                  class="plans-preview-inline text-pretty"
                  :class="{ 'is-empty': textPreview(row.development_indicator) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal('Индикатор Программы развития', row.development_indicator)"
                  @keyup.enter="openReadModal('Индикатор Программы развития', row.development_indicator)"
                  @keyup.space.prevent="openReadModal('Индикатор Программы развития', row.development_indicator)"
                >
                  <span class="plans-preview-content">{{ textPreview(row.development_indicator) }}</span>
                </div>
                <span class="planned-value-chip">
                  {{ formatPlannedValue(row.planned_value, row.measurement_unit || row.unit) }}
                </span>
              </td>

              <td data-label="Мероприятия по достижению индикатора">
                <div
                  class="plans-preview-inline text-pretty"
                  :class="{ 'is-empty': textPreview(row.activities) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal('Мероприятия по достижению индикатора', row.activities)"
                  @keyup.enter="openReadModal('Мероприятия по достижению индикатора', row.activities)"
                  @keyup.space.prevent="openReadModal('Мероприятия по достижению индикатора', row.activities)"
                >
                  <span class="plans-preview-content">{{ textPreview(row.activities) }}</span>
                </div>
              </td>

              <td data-label="Срок исполнения">
                <div class="plans-schedule-card">
                  <div class="plans-cell-frame">{{ formatDateRange(row) || '—' }}</div>
                  <div class="schedule-status" :class="`schedule-${row.schedule_status}`">
                    {{ scheduleStatusLabel(row.schedule_status) }}
                  </div>
                  <div v-if="row.execution_start_date && row.execution_end_date" class="countdown-text">
                    Қалған уақыт: {{ formatRemainingTime(row) }}
                  </div>
                </div>
              </td>

              <td data-label="Ответственные">
                <template v-if="isAdmin">
                  <div class="plans-cell-frame responsible-preview text-pretty">
                    {{ row.responsible || 'Ответственные таңдалмаған' }}
                  </div>
                  <button
                    class="btn btn-primary plans-edit-row-btn"
                    type="button"
                    @click="openRowEditModal(row)"
                  >
                    Өзгерту
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

    <div v-if="readModalOpen" class="modal-backdrop" @click.self="closeReadModal">
      <div class="modal-card plans-modal plans-read-modal">
        <h3 class="modal-title">{{ readModalTitle }}</h3>
        <div class="plans-read-content text-pretty">
          {{ readModalText }}
        </div>

        <div class="modal-actions">
          <button class="btn btn-primary" type="button" @click="closeReadModal">
            Жабу
          </button>
        </div>
      </div>
    </div>

    <div v-if="rowEditModalOpen" class="modal-backdrop" @click.self="closeRowEditModal">
      <div class="modal-card plans-modal plans-row-edit-modal">
        <h3 class="modal-title">Индикаторды өзгерту</h3>
        <p class="modal-subtitle">
          {{ activeRowEdit?.development_indicator || 'Индикатор' }}
        </p>

        <div class="row-edit-grid">
          <label class="modal-label">
            Индикатор Программы развития
            <textarea
              v-model="rowEditForm.development_indicator"
              class="plans-edit-textarea"
              rows="5"
              placeholder="Индикатор мәтінін жазыңыз..."
            />
          </label>

          <label class="modal-label">
            Мероприятия по достижению индикатора
            <textarea
              v-model="rowEditForm.activities"
              class="plans-edit-textarea"
              rows="8"
              placeholder="Мероприятия мәтінін жазыңыз..."
            />
          </label>

          <div class="row-edit-dates">
            <div class="date-range-grid">
              <label class="date-field">
                <span>Басталуы</span>
                <input v-model="rowEditForm.execution_start_date" class="plans-input" type="date" />
              </label>
              <label class="date-field">
                <span>Аяқталуы</span>
                <input v-model="rowEditForm.execution_end_date" class="plans-input" type="date" />
              </label>
            </div>
            <p class="date-range-preview">
              {{ formatDateRange(rowEditForm) || '—' }}
            </p>
          </div>

          <label class="modal-label">
            Ответственные
            <div class="prorector-list row-edit-prorectors">
              <label
                v-for="prorector in prorectors"
                :key="`row-edit-prorector-${prorector.id}`"
                class="prorector-item"
              >
                <input
                  v-model="rowEditForm.responsible_user_ids"
                  type="checkbox"
                  :value="Number(prorector.id)"
                />
                <span>
                  <strong>{{ prorector.full_name }}</strong>
                  <small>{{ prorector.username }}</small>
                </span>
              </label>
              <p v-if="prorectors.length === 0" class="empty-state">
                Проректорлар тізімі жоқ.
              </p>
            </div>
          </label>
        </div>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeRowEditModal">
            Бас тарту
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="savingIndicatorId === rowEditIndicatorId"
            @click="saveRowFromModal"
          >
            {{ savingIndicatorId === rowEditIndicatorId ? 'Сақталуда...' : 'Сақтау' }}
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

.plans-preview-inline {
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.plans-preview-inline:focus-visible {
  outline: 2px solid rgba(17, 120, 111, 0.5);
  outline-offset: 4px;
  border-radius: 8px;
}

.plans-preview-content {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
  line-height: 1.48;
  word-break: break-word;
  overflow: hidden;
  -webkit-mask-image: linear-gradient(180deg, #000 72%, transparent);
  mask-image: linear-gradient(180deg, #000 72%, transparent);
}

.plans-preview-inline.is-empty .plans-preview-content {
  color: var(--muted);
  -webkit-mask-image: none;
  mask-image: none;
}

.plans-cell-frame {
  min-height: 5rem;
  padding: 0.9rem;
  border-radius: 18px;
  border: 1px solid rgba(16, 33, 42, 0.08);
  background: rgba(255, 255, 255, 0.68);
}

.planned-value-chip {
  display: inline-flex;
  align-items: center;
  margin-top: 0.6rem;
  border: 1px solid #d8e0ea;
  border-radius: 999px;
  padding: 0.2rem 0.58rem;
  color: #475569;
  background: #f8fafc;
  font-size: 0.78rem;
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
  min-height: 5.2rem;
}

.plans-edit-row-btn,
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

.plans-row-edit-modal {
  width: min(920px, 100%);
}

.plans-read-modal {
  width: min(760px, 100%);
}

.plans-read-content {
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

.plans-edit-textarea {
  width: 100%;
  min-height: 9.2rem;
  resize: vertical;
}

.row-edit-grid {
  display: grid;
  gap: 0.72rem;
}

.row-edit-dates {
  display: grid;
  gap: 0.55rem;
}

.row-edit-prorectors {
  max-height: 14rem;
  overflow: auto;
  padding-right: 0.2rem;
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

  .plans-filter {
    min-width: 100%;
  }
}

@media (max-width: 980px) {
  .plans-table-card {
    padding: 0.9rem;
  }

  .table-wrapper {
    overflow: visible;
    border: 0;
    box-shadow: none;
    background: transparent;
  }

  .plan-table {
    min-width: 0;
    display: block;
  }

  .plan-table thead {
    display: none;
  }

  .plan-table tbody {
    display: grid;
    gap: 0.85rem;
  }

  .plan-table tbody tr {
    display: block;
    padding: 0.7rem 0.85rem;
    border: 1px solid var(--border);
    border-radius: 20px;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: var(--shadow-soft);
  }

  .plan-table tbody td {
    display: grid;
    grid-template-columns: minmax(130px, 38%) 1fr;
    gap: 0.6rem;
    padding: 0.52rem 0.1rem;
    border-bottom: 1px dashed rgba(16, 33, 42, 0.12);
  }

  .plan-table tbody td:last-child {
    border-bottom: 0;
    padding-bottom: 0.2rem;
  }

  .plan-table tbody td::before {
    content: attr(data-label);
    color: var(--muted);
    font-size: 0.74rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .number-cell {
    align-items: center;
    justify-content: start;
    text-align: left;
    font-size: 1rem;
  }

  .plans-cell-frame {
    min-height: auto;
    padding: 0.72rem;
    border-radius: 14px;
  }

  .date-range-grid {
    grid-template-columns: 1fr;
  }

  .plans-edit-row-btn,
  .plans-report-btn {
    margin-top: 0.52rem;
  }
}

@media (max-width: 640px) {
  .plans-year-card strong,
  .plans-visible-card strong {
    font-size: 2rem;
  }

  .panel-header {
    gap: 0.7rem;
  }

  .plan-table tbody td {
    grid-template-columns: 1fr;
    gap: 0.42rem;
  }

  .plan-table tbody td::before {
    font-size: 0.7rem;
  }
}
</style>
