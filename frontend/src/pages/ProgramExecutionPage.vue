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

function canReviewRow(row) {
  const normalized = String(row?.status ?? '').toLowerCase()
  return normalized === 'pending'
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
      eyebrow="Reports"
    />

    <div class="panel panel-strong toolbar-panel execution-toolbar">
      <label class="execution-year-picker">
        <span>Год</span>
        <select :value="selectedYear" :disabled="loading || !hasYears" @change="handleYearChange">
          <option v-if="!hasYears" value="">Нет годов</option>
          <option v-for="year in years" :key="year" :value="String(year)">
            {{ year }}
          </option>
        </select>
      </label>

      <div v-if="isProrector" class="execution-categories">
        <button
          class="btn btn-ghost execution-category-btn"
          :class="{ 'is-active': prorectorCategory === 'pending' }"
          type="button"
          @click="handleProrectorCategoryChange('pending')"
        >
          На проверке
        </button>
        <button
          class="btn btn-ghost execution-category-btn"
          :class="{ 'is-active': prorectorCategory === 'rejected' }"
          type="button"
          @click="handleProrectorCategoryChange('rejected')"
        >
          Rejected
        </button>
      </div>

      <div class="execution-summary-card">
        <span class="kicker">Rows</span>
        <strong>{{ rows.length }}</strong>
      </div>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <section class="panel panel-strong execution-table-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">Отчеты и решения</h3>
          <p class="panel-subtitle">Файлы, текст отчета, формулы проверки и причина отклонения видны в одном потоке.</p>
        </div>
        <span class="kicker">{{ selectedYear || 'No year' }}</span>
      </div>

      <div v-if="loading" class="empty-state">Загрузка...</div>
      <div v-else-if="!hasYears" class="empty-state">
        Әзірге жылдар жоқ.
      </div>
      <div v-else-if="rows.length === 0" class="empty-state">
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
        <table class="table execution-table">
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
              <td class="text-pretty">{{ row.development_indicator || '—' }}</td>
              <td>{{ formatPlannedValue(row.planned_value, row.unit) }}</td>
              <td>{{ row.execution_deadline || '—' }}</td>
              <td class="text-pretty">{{ row.responsible || '—' }}</td>
              <td>
                <div class="report-card">
                  <p class="report-text">{{ row.report_text || '—' }}</p>
                  <div class="files-list">
                    <button
                      v-for="file in row.files"
                      :key="file.id"
                      class="btn btn-ghost report-file-chip"
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
                  <p v-if="row.approval_formula" class="formula-text text-pretty">
                    Формула: {{ row.approval_formula }}
                  </p>
                  <p v-if="row.review_note" class="reject-note text-pretty">
                    Причина: {{ row.review_note }}
                  </p>
                </div>
              </td>
              <td>
                <span class="status-pill" :class="`status-${row.status}`">
                  {{ statusLabel(row.status) }}
                </span>
              </td>
              <td class="actions-cell">
                <template v-if="canReviewRow(row)">
                  <button class="btn btn-primary action-btn" type="button" @click="openApproveModal(row)">
                    Қабылдау
                  </button>
                  <button class="btn btn-danger action-btn" type="button" @click="openRejectModal(row)">
                    Қабылдамау
                  </button>
                </template>
                <span v-else class="muted">Қаралған</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else class="table-wrap">
        <table class="table execution-table execution-table-prorector">
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
              <td class="text-pretty">{{ row.development_indicator || '—' }}</td>
              <td>{{ formatPlannedValue(row.planned_value, row.unit) }}</td>
              <td>{{ row.execution_deadline || '—' }}</td>
              <td class="text-pretty">{{ row.responsible || '—' }}</td>
              <td v-if="isRejectedCategory" class="text-pretty">{{ row.review_note || '—' }}</td>
              <td v-else>
                <span class="status-pill status-pending">На проверке</span>
              </td>
              <td>
                <div class="report-card">
                  <p class="report-text">{{ row.report_text || '—' }}</p>
                  <p class="meta">
                    Отправлено: {{ formatDate(row.submitted_at) }}
                  </p>
                </div>
              </td>
              <td>
                <div class="files-list">
                  <button
                    v-for="file in row.files"
                    :key="file.id"
                    class="btn btn-ghost report-file-chip"
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
                  class="btn btn-primary action-btn"
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
    </section>

    <div v-if="reviewModalOpen" class="modal-backdrop" @click.self="closeReviewModal">
      <div class="modal-card execution-modal">
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
          <button class="btn btn-ghost" type="button" @click="closeReviewModal">
            Бас тарту
          </button>
          <button
            class="btn btn-primary"
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
      <div class="modal-card execution-modal">
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
          <input
            class="file-input"
            type="file"
            accept=".doc,.docx,.xls,.xlsx,.ppt,.pptx,.pdf"
            multiple
            @change="handleResubmitFiles"
          />
        </label>
        <p v-if="resubmitFiles.length > 0" class="file-info">
          Таңдалған файлдар: {{ resubmitFiles.map((file) => file.name).join(', ') }}
        </p>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeResubmitModal">
            Бас тарту
          </button>
          <button
            class="btn btn-primary"
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
.execution-toolbar {
  justify-content: space-between;
}

.execution-year-picker {
  min-width: 13rem;
}

.execution-categories {
  display: flex;
  flex-wrap: wrap;
  gap: 0.7rem;
}

.execution-category-btn.is-active {
  background: linear-gradient(135deg, var(--accent), var(--accent-strong));
  color: #fffdf9;
  border-color: transparent;
}

.execution-summary-card {
  display: grid;
  gap: 0.35rem;
}

.execution-summary-card strong {
  font-size: clamp(2rem, 3vw, 2.7rem);
  line-height: 0.95;
  letter-spacing: -0.05em;
}

.execution-table-card {
  padding: 1.2rem;
}

.execution-table {
  min-width: 1340px;
}

.number-cell {
  width: 4rem;
  text-align: center;
  font-weight: 700;
}

.report-card {
  display: grid;
  gap: 0.55rem;
}

.report-text {
  margin: 0;
  white-space: pre-wrap;
}

.meta {
  margin: 0;
  color: var(--muted);
  font-size: 0.82rem;
}

.formula-text {
  margin: 0;
  color: #1b6fa8;
  font-size: 0.84rem;
}

.reject-note {
  margin: 0;
  color: #a63f32;
  font-size: 0.84rem;
}

.files-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.report-file-chip {
  min-height: auto;
  padding: 0.45rem 0.8rem;
  font-size: 0.82rem;
}

.actions-cell {
  min-width: 12rem;
}

.action-btn {
  width: 100%;
  justify-content: center;
}

.action-btn + .action-btn {
  margin-top: 0.65rem;
}

.execution-modal {
  width: min(720px, 100%);
}

.modal-label {
  display: grid;
  gap: 0.45rem;
  margin-top: 1rem;
}

.modal-textarea,
.file-input {
  width: 100%;
}

.file-info {
  margin: 0.75rem 0 0;
  color: var(--muted);
  font-size: 0.88rem;
}

@media (max-width: 1024px) {
  .execution-toolbar {
    align-items: stretch;
  }
}
</style>
