<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import {
  downloadPlanReportFile,
  fetchPlanReports,
  fetchPlanYears,
  reviewPlanReport,
  submitPlanIndicatorReport
} from '../services/plan.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()

const years = ref([])
const selectedYear = ref('')
const rows = ref([])
const loading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const prorectorCategory = ref('pending')

const reviewModalOpen = ref(false)
const reviewMode = ref('approve')
const reviewRowId = ref(null)
const reviewText = ref('')
const reviewSaving = ref(false)

const resubmitModalOpen = ref(false)
const resubmitRow = ref(null)
const resubmitText = ref('')
const resubmitFiles = ref([])
const resubmitSending = ref(false)

const isAdmin = computed(() => authStore.user?.role === 'admin')
const isProrector = computed(() => authStore.user?.role === 'prorector')
const isRejectedCategory = computed(() => prorectorCategory.value === 'rejected')
const hasYears = computed(() => years.value.length > 0)
const canLoadYear = computed(() => selectedYear.value !== '')
const activeReviewRow = computed(() => rows.value.find((item) => item.id === reviewRowId.value) ?? null)
const pageSubtitle = computed(() => {
  if (isAdmin.value) {
    return 'Проректорлар жіберген индикаторлар, файлдар және қабылдау/қабылдамау'
  }
  if (isProrector.value) {
    if (isRejectedCategory.value) {
      return 'Rejected отчеттар. Түзетіп, қайта жіберуге болады'
    }
    return 'На проверке: админнің жауабын күтіп тұрған отчеттар'
  }
  return 'Отчеттар тізімі'
})

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
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

function statusLabel(status) {
  const normalized = String(status ?? '').toLowerCase()
  if (normalized === 'completed' || normalized === 'approved') {
    return 'Completed'
  }
  if (normalized === 'rejected') {
    return 'Rejected'
  }
  if (normalized === 'overdue') {
    return 'Overdue'
  }
  return 'Pending'
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

  const options = {}
  if (isProrector.value) {
    options.status = isRejectedCategory.value ? 'rejected' : 'pending'
  } else if (isAdmin.value) {
    options.status = 'pending,rejected'
  }

  const response = await fetchPlanReports(selectedYear.value, options)
  rows.value = response.items ?? []
}

async function initialize() {
  loading.value = true
  clearMessages()

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
  clearMessages()

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

async function handleProrectorCategoryChange(category) {
  if (!isProrector.value) {
    return
  }

  prorectorCategory.value = category
  loading.value = true
  clearMessages()

  try {
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Категория бойынша мәліметтерді жүктеу мүмкін болмады'
  } finally {
    loading.value = false
  }
}

function parseContentDispositionFilename(contentDisposition) {
  if (!contentDisposition) {
    return ''
  }

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1])
    } catch {
      return utf8Match[1]
    }
  }

  const plainMatch = contentDisposition.match(/filename="?([^"]+)"?/i)
  if (plainMatch?.[1]) {
    return plainMatch[1]
  }

  return ''
}

async function extractErrorMessage(error, fallback) {
  const rawData = error?.response?.data

  if (typeof rawData === 'string' && rawData.trim() !== '') {
    return rawData
  }

  if (rawData instanceof Blob) {
    try {
      const text = await rawData.text()
      if (text) {
        try {
          const parsed = JSON.parse(text)
          if (parsed?.error) {
            return parsed.error
          }
        } catch {
          return text
        }
      }
    } catch {
      // ignore blob parsing errors
    }
  }

  if (error?.response?.data?.error) {
    return error.response.data.error
  }
  if (error?.message) {
    return error.message
  }

  return fallback
}

async function handleDownload(file) {
  clearMessages()

  try {
    const response = await downloadPlanReportFile(file.id)
    const blobUrl = window.URL.createObjectURL(response.data)
    const disposition = response.headers?.['content-disposition'] ?? ''
    const filename = parseContentDispositionFilename(disposition) || file.file_name || `report-${file.id}`

    const link = document.createElement('a')
    link.href = blobUrl
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(blobUrl)
  } catch (error) {
    errorMessage.value = await extractErrorMessage(error, 'Файлды жүктеу мүмкін болмады')
  }
}

function openApproveModal(row) {
  reviewMode.value = 'approve'
  reviewRowId.value = row.id
  reviewText.value = row.approval_formula ?? ''
  reviewModalOpen.value = true
  clearMessages()
}

function openRejectModal(row) {
  reviewMode.value = 'reject'
  reviewRowId.value = row.id
  reviewText.value = row.review_note ?? ''
  reviewModalOpen.value = true
  clearMessages()
}

function closeReviewModal() {
  reviewModalOpen.value = false
  reviewRowId.value = null
  reviewText.value = ''
}

async function submitReviewDecision() {
  if (!isAdmin.value || reviewRowId.value === null) {
    return
  }

  const normalizedText = reviewText.value.trim()
  if (!normalizedText) {
    errorMessage.value = reviewMode.value === 'approve'
      ? 'Формуланы толтырыңыз'
      : 'Қабылдамау себебін толтырыңыз'
    return
  }

  reviewSaving.value = true
  clearMessages()

  try {
    if (reviewMode.value === 'approve') {
      await reviewPlanReport(reviewRowId.value, {
        action: 'approve',
        approval_formula: normalizedText
      })
      successMessage.value = 'Отчет қабылданды'
    } else {
      await reviewPlanReport(reviewRowId.value, {
        action: 'reject',
        review_note: normalizedText
      })
      successMessage.value = 'Отчет қабылданбады'
    }

    closeReviewModal()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Статусты жаңарту мүмкін болмады'
  } finally {
    reviewSaving.value = false
  }
}

function openResubmitModal(row) {
  if (!isProrector.value) {
    return
  }

  resubmitRow.value = row
  resubmitText.value = row.report_text ?? ''
  resubmitFiles.value = []
  resubmitModalOpen.value = true
  clearMessages()
}

function closeResubmitModal() {
  resubmitModalOpen.value = false
  resubmitRow.value = null
  resubmitText.value = ''
  resubmitFiles.value = []
}

function handleResubmitFiles(event) {
  const files = event?.target?.files ?? []
  resubmitFiles.value = Array.from(files).filter(Boolean)
}

async function submitResubmittedReport() {
  if (!isProrector.value || !resubmitRow.value || !canLoadYear.value) {
    return
  }

  if (resubmitFiles.value.length === 0) {
    errorMessage.value = 'Кемінде бір құжат жүктеу міндетті'
    return
  }

  resubmitSending.value = true
  clearMessages()

  try {
    await submitPlanIndicatorReport(resubmitRow.value.indicator_id, selectedYear.value, {
      report_text: resubmitText.value.trim(),
      files: resubmitFiles.value
    })
    successMessage.value = 'Отчет қайта жіберілді'
    closeResubmitModal()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? 'Отчетты қайта жіберу мүмкін болмады'
  } finally {
    resubmitSending.value = false
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
      :subtitle="pageSubtitle"
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

      <div v-if="isProrector" class="prorector-categories">
        <button
          class="category-btn"
          :class="{ 'is-active': prorectorCategory === 'pending' }"
          type="button"
          @click="handleProrectorCategoryChange('pending')"
        >
          На проверке
        </button>
        <button
          class="category-btn"
          :class="{ 'is-active': prorectorCategory === 'rejected' }"
          type="button"
          @click="handleProrectorCategoryChange('rejected')"
        >
          Rejected
        </button>
      </div>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div v-if="loading" class="state-box">Загрузка...</div>
    <div v-else-if="!hasYears" class="state-box">
      Әзірге жылдар жоқ.
    </div>
    <div v-else-if="rows.length === 0" class="state-box">
      <template v-if="isProrector">
        <template v-if="isRejectedCategory">
          {{ selectedYear }} жылына Rejected отчеттар жоқ.
        </template>
        <template v-else>
          {{ selectedYear }} жылына На проверке категориясында отчеттар жоқ.
        </template>
      </template>
      <template v-else>
        {{ selectedYear }} жылына жіберілген отчеттар жоқ.
      </template>
    </div>

    <div v-else-if="isAdmin" class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>№</th>
            <th>Целевой индикатор</th>
            <th>Мән</th>
            <th>Срок исполнения</th>
            <th>Ответственные</th>
            <th>Выполнение индикатора</th>
            <th>Статус</th>
            <th>Решение</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in rows" :key="row.id">
            <td class="number-cell">{{ index + 1 }}</td>
            <td>{{ row.development_indicator || '—' }}</td>
            <td>{{ formatPlannedValue(row.planned_value, row.unit) }}</td>
            <td>{{ row.execution_deadline || '—' }}</td>
            <td>{{ row.responsible || '—' }}</td>
            <td>
              <p class="report-text">{{ row.report_text || '—' }}</p>
              <div class="files-list">
                <button
                  v-for="file in row.files"
                  :key="file.id"
                  class="file-download-btn"
                  type="button"
                  @click="handleDownload(file)"
                >
                  {{ file.file_name }}
                </button>
                <span v-if="!row.files || row.files.length === 0" class="muted">Файл жоқ</span>
              </div>
              <p class="meta">
                Отправил: {{ row.submitted_by_name || row.submitted_by }} • {{ formatDate(row.submitted_at) }}
              </p>
              <p v-if="row.reviewed_by_name" class="meta">
                Проверил: {{ row.reviewed_by_name }} • {{ formatDate(row.reviewed_at) }}
              </p>
              <p v-if="row.approval_formula" class="formula-text">
                Формула: {{ row.approval_formula }}
              </p>
              <p v-if="row.review_note" class="reject-note">
                Причина: {{ row.review_note }}
              </p>
            </td>
            <td>
              <span class="status-pill" :class="`status-${row.status}`">
                {{ statusLabel(row.status) }}
              </span>
            </td>
            <td class="actions-cell">
              <button class="action-btn approve-btn" type="button" @click="openApproveModal(row)">
                Қабылдау
              </button>
              <button class="action-btn reject-btn" type="button" @click="openRejectModal(row)">
                Қабылдамау
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>№</th>
            <th>Целевой индикатор</th>
            <th>Мән</th>
            <th>Срок исполнения</th>
            <th>Ответственные</th>
            <th v-if="isRejectedCategory">Причина Rejected</th>
            <th v-else>Статус</th>
            <th>Алдыңғы отчет</th>
            <th>Құжаттар</th>
            <th>Әрекет</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in rows" :key="row.id">
            <td class="number-cell">{{ index + 1 }}</td>
            <td>{{ row.development_indicator || '—' }}</td>
            <td>{{ formatPlannedValue(row.planned_value, row.unit) }}</td>
            <td>{{ row.execution_deadline || '—' }}</td>
            <td>{{ row.responsible || '—' }}</td>
            <td v-if="isRejectedCategory">{{ row.review_note || '—' }}</td>
            <td v-else>
              <span class="status-pill status-pending">На проверке</span>
            </td>
            <td>
              <p class="report-text">{{ row.report_text || '—' }}</p>
              <p class="meta">
                Отправлено: {{ formatDate(row.submitted_at) }}
              </p>
            </td>
            <td>
              <div class="files-list">
                <button
                  v-for="file in row.files"
                  :key="file.id"
                  class="file-download-btn"
                  type="button"
                  @click="handleDownload(file)"
                >
                  {{ file.file_name }}
                </button>
                <span v-if="!row.files || row.files.length === 0" class="muted">Файл жоқ</span>
              </div>
            </td>
            <td>
              <button
                v-if="isRejectedCategory"
                class="action-btn resend-btn"
                type="button"
                @click="openResubmitModal(row)"
              >
                Исправить и отправить
              </button>
              <span v-else class="muted">Күтілуде</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="reviewModalOpen" class="modal-backdrop" @click.self="closeReviewModal">
      <div class="modal-card">
        <h3 class="modal-title">
          {{ reviewMode === 'approve' ? 'Отчетты қабылдау' : 'Отчетты қабылдамау' }}
        </h3>
        <p class="modal-subtitle">
          {{ activeReviewRow?.development_indicator || 'Индикатор' }}
        </p>

        <label class="modal-label">
          <span v-if="reviewMode === 'approve'">
            Формула және қорытынды сан
          </span>
          <span v-else>
            Неге қабылданбады
          </span>
          <textarea
            v-model="reviewText"
            class="modal-textarea"
            rows="6"
            :placeholder="reviewMode === 'approve'
              ? 'Мысалы: 157/525 ППС + 11587 обучающихся *100%=1,3%'
              : 'Қабылданбау себебін жазыңыз...'"
          />
        </label>

        <div class="modal-actions">
          <button class="modal-btn modal-btn-secondary" type="button" @click="closeReviewModal">
            Бас тарту
          </button>
          <button
            class="modal-btn modal-btn-primary"
            type="button"
            :disabled="reviewSaving"
            @click="submitReviewDecision"
          >
            {{ reviewSaving ? 'Сақталуда...' : (reviewMode === 'approve' ? 'Жабу' : 'Отправить') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="resubmitModalOpen" class="modal-backdrop" @click.self="closeResubmitModal">
      <div class="modal-card">
        <h3 class="modal-title">Rejected отчетты қайта жіберу</h3>
        <p class="modal-subtitle">
          {{ resubmitRow?.development_indicator || 'Индикатор' }}
        </p>

        <label class="modal-label">
          Отчет мәтіні
          <textarea
            v-model="resubmitText"
            class="modal-textarea"
            rows="6"
            placeholder="Түзетілген отчетты жазыңыз..."
          />
        </label>

        <label class="modal-label">
          Құжаттар (кемінде 1 файл)
          <input class="file-input" type="file" multiple @change="handleResubmitFiles" />
        </label>
        <p v-if="resubmitFiles.length > 0" class="file-info">
          Таңдалған файлдар: {{ resubmitFiles.map((file) => file.name).join(', ') }}
        </p>

        <div class="modal-actions">
          <button class="modal-btn modal-btn-secondary" type="button" @click="closeResubmitModal">
            Бас тарту
          </button>
          <button
            class="modal-btn modal-btn-primary"
            type="button"
            :disabled="resubmitSending"
            @click="submitResubmittedReport"
          >
            {{ resubmitSending ? 'Жіберілуде...' : 'Отправить' }}
          </button>
        </div>
      </div>
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
  align-items: center;
  gap: 0.8rem;
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

.prorector-categories {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
}

.category-btn {
  border: 1px solid #cbd5e1;
  border-radius: 999px;
  padding: 0.38rem 0.8rem;
  background: #ffffff;
  color: #334155;
  font-size: 0.85rem;
  font-weight: 700;
  cursor: pointer;
}

.category-btn.is-active {
  border-color: #0f766e;
  background: #0f766e;
  color: #ffffff;
}

.table-wrap {
  overflow-x: auto;
  border: 1px solid #d8e0ea;
  border-radius: 10px;
  background: #ffffff;
}

.table {
  width: 100%;
  min-width: 1320px;
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
  margin: 0;
  white-space: pre-wrap;
}

.meta {
  margin: 0.35rem 0 0;
  font-size: 0.8rem;
  color: #64748b;
}

.formula-text {
  margin: 0.4rem 0 0;
  font-size: 0.82rem;
  color: #1d4ed8;
  white-space: pre-wrap;
}

.reject-note {
  margin: 0.4rem 0 0;
  font-size: 0.82rem;
  color: #b91c1c;
  white-space: pre-wrap;
}

.files-list {
  margin-top: 0.45rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.file-download-btn {
  border: 1px solid #334155;
  border-radius: 6px;
  padding: 0.3rem 0.55rem;
  background: #f8fafc;
  color: #0f172a;
  font-size: 0.82rem;
  cursor: pointer;
}

.muted {
  color: #64748b;
  font-size: 0.82rem;
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

.status-approved {
  background: #dcfce7;
  color: #166534;
}

.status-rejected {
  background: #fee2e2;
  color: #991b1b;
}

.actions-cell {
  min-width: 160px;
}

.action-btn {
  width: 100%;
  border-radius: 7px;
  padding: 0.45rem 0.6rem;
  font-weight: 700;
  cursor: pointer;
}

.action-btn + .action-btn {
  margin-top: 0.45rem;
}

.approve-btn {
  border: 1px solid #166534;
  background: #166534;
  color: #ffffff;
}

.reject-btn {
  border: 1px solid #b91c1c;
  background: #ffffff;
  color: #b91c1c;
}

.resend-btn {
  border: 1px solid #0f766e;
  background: #0f766e;
  color: #ffffff;
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
  width: min(700px, 100%);
  border-radius: 12px;
  background: #ffffff;
  border: 1px solid #d8e0ea;
  box-shadow: 0 18px 40px rgba(2, 6, 23, 0.2);
  padding: 1rem;
}

.modal-title {
  margin: 0;
  font-size: 1.08rem;
}

.modal-subtitle {
  margin: 0.4rem 0 0.7rem;
  color: #475569;
  font-size: 0.9rem;
}

.modal-label {
  display: grid;
  gap: 0.35rem;
  margin-top: 0.6rem;
  font-size: 0.9rem;
}

.modal-textarea,
.file-input {
  width: 100%;
  border: 1px solid #c8d2de;
  border-radius: 8px;
  padding: 0.5rem 0.6rem;
  font: inherit;
  background: #fff;
}

.file-info {
  margin: 0.3rem 0 0;
  font-size: 0.82rem;
  color: #475569;
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

@media (max-width: 920px) {
  .modal-card {
    max-height: 90vh;
    overflow-y: auto;
  }
}
</style>
