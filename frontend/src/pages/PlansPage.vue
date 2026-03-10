<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import {
  fetchPlanIndicators,
  fetchPlanYears,
  savePlanIndicator,
  submitPlanIndicatorReport
} from '../services/plan.service'
import { fetchProrectorsRequest } from '../services/user.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()

const years = ref([])
const selectedYear = ref('')
const rows = ref([])
const prorectors = ref([])
const loading = ref(false)
const savingIndicatorId = ref(null)
const errorMessage = ref('')
const successMessage = ref('')
const assignModalOpen = ref(false)
const activeIndicatorId = ref(null)
const modalSelectedIds = ref([])
const reportModalOpen = ref(false)
const reportIndicatorId = ref(null)
const reportText = ref('')
const reportFile = ref(null)
const reportSending = ref(false)

const isAdmin = computed(() => authStore.user?.role === 'admin')
const isProrector = computed(() => authStore.user?.role === 'prorector')
const hasYears = computed(() => years.value.length > 0)
const canLoadYear = computed(() => selectedYear.value !== '')
const activeReportRow = computed(() => rows.value.find((item) => item.indicator_id === reportIndicatorId.value) ?? null)

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

  const response = await fetchPlanIndicators(selectedYear.value)
  rows.value = (response.items ?? []).map((item) => ({
    ...item,
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
    await loadYears()
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

async function handleYearChange(event) {
  selectedYear.value = event.target.value
  clearMessages()
  loading.value = true

  try {
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Жыл бойынша мәліметтерді жүктеу кезінде қате болды'
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
    const saved = await savePlanIndicator(row.indicator_id, selectedYear.value, {
      development_indicator: row.development_indicator,
      activities: row.activities,
      execution_deadline: row.execution_deadline,
      responsible_user_ids: normalizeIDList(row.responsible_user_ids)
    })

    Object.assign(row, {
      ...saved,
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
  reportFile.value = null
  reportModalOpen.value = true
}

function closeReportModal() {
  reportModalOpen.value = false
  reportIndicatorId.value = null
  reportText.value = ''
  reportFile.value = null
}

function handleReportFileChange(event) {
  const [file] = event?.target?.files ?? []
  reportFile.value = file || null
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
  if (!normalizedText && !reportFile.value) {
    errorMessage.value = 'Отчет мәтінін енгізіңіз немесе файл тіркеңіз'
    return
  }

  reportSending.value = true
  clearMessages()

  try {
    await submitPlanIndicatorReport(row.indicator_id, selectedYear.value, {
      report_text: normalizedText,
      file: reportFile.value
    })
    successMessage.value = `Индикатор №${row.indicator_id} бойынша отчет жіберілді`
    closeReportModal()
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
  initialize()
})
</script>

<template>
  <section class="plans-page">
    <PageHeader
      title="Plans"
      subtitle="Индикаторы и мероприятия по выбранному году из Planning Period"
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
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div v-if="loading" class="state-box">Загрузка...</div>
    <div v-else-if="!hasYears" class="state-box">
      Planning Period бөлімінде әлі жылдық дерек жоқ. Алдымен сол бөлімде индикатор енгізіңіз.
    </div>
    <div v-else-if="rows.length === 0" class="state-box">
      {{ selectedYear }} жылына индикатор табылмады.
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
          <tr v-for="(row, index) in rows" :key="row.indicator_id">
            <td class="number-cell">{{ index + 1 }}</td>

            <td>
              <template v-if="isAdmin">
                <textarea
                  v-model="row.development_indicator"
                  class="cell-textarea indicator-text"
                  rows="3"
                />
                <div class="unit-inline">
                  {{ formatPlannedValue(row.planned_value, row.measurement_unit || row.unit) }}
                </div>
              </template>
              <template v-else>
                <div class="cell-readonly">{{ row.development_indicator }}</div>
                <div class="unit-inline">
                  {{ formatPlannedValue(row.planned_value, row.measurement_unit || row.unit) }}
                </div>
              </template>
            </td>

            <td>
              <template v-if="isAdmin">
                <textarea v-model="row.activities" class="cell-textarea" rows="4" />
              </template>
              <template v-else>
                <div class="cell-readonly">{{ row.activities || '—' }}</div>
              </template>
            </td>

            <td>
              <template v-if="isAdmin">
                <input v-model="row.execution_deadline" class="cell-input" type="text" />
              </template>
              <template v-else>
                <div class="cell-readonly">{{ row.execution_deadline || '—' }}</div>
              </template>
            </td>

            <td>
              <template v-if="isAdmin">
                <div class="cell-readonly responsible-preview">
                  {{ row.responsible || 'Ответственные таңдалмаған' }}
                </div>
                <button class="assign-btn" type="button" @click="openResponsibleModal(row)">
                  Ответственные бекіту
                </button>
                <p v-if="prorectors.length === 0" class="cell-note">
                  Проректорлар тізімі жоқ.
                </p>
                <button
                  class="save-btn"
                  :disabled="savingIndicatorId === row.indicator_id"
                  @click="saveRow(row)"
                >
                  {{ savingIndicatorId === row.indicator_id ? 'Сақталуда...' : 'Сақтау' }}
                </button>
              </template>
              <template v-else>
                <div class="cell-readonly">{{ row.responsible || '—' }}</div>
                <button
                  v-if="isProrector"
                  class="report-btn"
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

    <div v-if="assignModalOpen" class="modal-backdrop" @click.self="closeResponsibleModal">
      <div class="modal-card">
        <h3 class="modal-title">Ответственные таңдау</h3>
        <p class="modal-subtitle">
          Керек проректорларға галочка қойып, «Бекіту» батырмасын басыңыз.
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
            <span>{{ prorector.full_name }} ({{ prorector.username }})</span>
          </label>

          <p v-if="prorectors.length === 0" class="empty-prorectors">
            Проректорлар тізімі бос.
          </p>
        </div>

        <div class="modal-actions">
          <button class="modal-btn modal-btn-secondary" type="button" @click="closeResponsibleModal">
            Бас тарту
          </button>
          <button class="modal-btn modal-btn-primary" type="button" @click="applyResponsibleSelection">
            Бекіту
          </button>
        </div>
      </div>
    </div>

    <div v-if="reportModalOpen" class="modal-backdrop" @click.self="closeReportModal">
      <div class="modal-card">
        <h3 class="modal-title">Отправить отчет</h3>
        <p class="modal-subtitle">
          {{ activeReportRow?.development_indicator || 'Индикатор' }}
        </p>

        <label class="report-label">
          Текст отчета
          <textarea
            v-model="reportText"
            class="report-textarea"
            rows="6"
            placeholder="Индикатор бойынша орындалу нәтижесін жазыңыз..."
          />
        </label>

        <label class="report-label">
          Файл (міндетті емес)
          <input
            class="report-file-input"
            type="file"
            @change="handleReportFileChange"
          />
        </label>

        <div class="modal-actions">
          <button class="modal-btn modal-btn-secondary" type="button" @click="closeReportModal">
            Бас тарту
          </button>
          <button
            class="modal-btn modal-btn-primary"
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
.plans-page {
  display: grid;
  gap: 0.85rem;
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

.table-wrapper {
  overflow-x: auto;
  border: 1px solid #d8e0ea;
  border-radius: 10px;
  background: #ffffff;
}

.plan-table {
  width: 100%;
  border-collapse: collapse;
  min-width: 1050px;
}

.plan-table th,
.plan-table td {
  border: 1px solid #d8e0ea;
  padding: 0.6rem;
  vertical-align: top;
}

.plan-table thead th {
  background: #f4f6f8;
  text-align: center;
  font-weight: 700;
}

.col-number {
  width: 48px;
}

.col-deadline {
  width: 180px;
}

.col-responsible {
  width: 220px;
}

.number-cell {
  text-align: center;
  font-weight: 600;
}

.cell-textarea,
.cell-input {
  width: 100%;
  border: 1px solid #c8d2de;
  border-radius: 6px;
  padding: 0.45rem 0.55rem;
  font: inherit;
  resize: vertical;
  background: #fff;
}

.indicator-text {
  min-height: 86px;
}

.unit-inline {
  margin-top: 0.35rem;
  font-size: 0.84rem;
  color: #475569;
}

.cell-readonly {
  white-space: pre-wrap;
  color: #0f172a;
}

.responsible-preview {
  min-height: 96px;
  border: 1px solid #c8d2de;
  border-radius: 6px;
  padding: 0.45rem 0.55rem;
  background: #fff;
}

.assign-btn {
  margin-top: 0.45rem;
  width: 100%;
  border: 1px solid #0f766e;
  border-radius: 7px;
  background: #ffffff;
  color: #0f766e;
  padding: 0.45rem 0.6rem;
  font-weight: 600;
  cursor: pointer;
}

.report-btn {
  margin-top: 0.5rem;
  width: 100%;
  border: 1px solid #0f766e;
  border-radius: 7px;
  background: #0f766e;
  color: #ffffff;
  padding: 0.45rem 0.6rem;
  font-weight: 600;
  cursor: pointer;
}

.cell-note {
  margin: 0.45rem 0 0;
  font-size: 0.8rem;
  color: #64748b;
}

.save-btn {
  margin-top: 0.5rem;
  width: 100%;
  border: none;
  border-radius: 7px;
  background: #0f766e;
  color: #ffffff;
  padding: 0.45rem 0.6rem;
  font-weight: 600;
  cursor: pointer;
}

.save-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
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

.message-success {
  background: #dcfce7;
  color: #166534;
}

.state-box {
  border: 1px dashed #cbd5e1;
  border-radius: 10px;
  background: #f8fafc;
  padding: 1rem;
  color: #475569;
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.45);
  display: grid;
  place-items: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-card {
  width: min(620px, 100%);
  border-radius: 12px;
  background: #ffffff;
  border: 1px solid #d8e0ea;
  box-shadow: 0 18px 40px rgba(2, 6, 23, 0.2);
  padding: 1rem;
}

.modal-title {
  margin: 0;
  font-size: 1.1rem;
}

.modal-subtitle {
  margin: 0.4rem 0 0.75rem;
  color: #475569;
  font-size: 0.9rem;
}

.prorector-list {
  max-height: 280px;
  overflow-y: auto;
  border: 1px solid #d8e0ea;
  border-radius: 8px;
  padding: 0.55rem;
  display: grid;
  gap: 0.5rem;
}

.prorector-item {
  display: flex;
  align-items: flex-start;
  gap: 0.55rem;
  font-size: 0.95rem;
}

.empty-prorectors {
  margin: 0;
  color: #64748b;
}

.modal-actions {
  margin-top: 0.85rem;
  display: flex;
  justify-content: flex-end;
  gap: 0.55rem;
}

.modal-btn {
  border-radius: 7px;
  padding: 0.45rem 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.modal-btn-secondary {
  border: 1px solid #cbd5e1;
  background: #ffffff;
  color: #0f172a;
}

.modal-btn-primary {
  border: 1px solid #0f766e;
  background: #0f766e;
  color: #ffffff;
}

.report-label {
  display: grid;
  gap: 0.35rem;
  margin-top: 0.55rem;
  font-size: 0.9rem;
}

.report-textarea,
.report-file-input {
  width: 100%;
  border: 1px solid #c8d2de;
  border-radius: 8px;
  padding: 0.5rem 0.6rem;
  font: inherit;
  background: #fff;
}

@media (max-width: 900px) {
  .toolbar {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.55rem;
  }

  .modal-card {
    max-height: 90vh;
    overflow-y: auto;
  }

  .prorector-list {
    max-height: 42vh;
  }
}
</style>
