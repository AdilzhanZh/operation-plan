<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import {
  downloadPlanReportFile,
  fetchPlanReports,
  fetchPlanYears,
  reviewPlanReport,
  submitPlanIndicatorReport
} from '../services/plan.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()
const { tr } = useLocale()

const years = ref([])
const selectedYear = ref('')
const rows = ref([])
const loading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const prorectorCategory = ref('pending')
const yearTrackRef = ref(null)
const canYearScrollLeft = ref(false)
const canYearScrollRight = ref(false)
const readModalOpen = ref(false)
const readModalTitle = ref('')
const readModalText = ref('')

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
    return tr('Индикаторы, отправленные проректорами, файлы и решения по принятию/отклонению', 'Проректорлар жіберген индикаторлар, файлдар және қабылдау/қабылдамау')
  }
  if (isProrector.value) {
    if (isRejectedCategory.value) {
      return tr('Отклоненные отчеты. Можно исправить и отправить повторно', 'Қабылданбаған есептер. Түзетіп, қайта жіберуге болады')
    }
    return tr('На проверке: отчеты, ожидающие решения администратора', 'На проверке: админнің жауабын күтіп тұрған отчеттар')
  }
  return tr('Список отчетов', 'Отчеттар тізімі')
})

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

function updateYearScrollState() {
  const track = yearTrackRef.value
  if (!track) {
    canYearScrollLeft.value = false
    canYearScrollRight.value = false
    return
  }

  const maxScroll = Math.max(0, track.scrollWidth - track.clientWidth)
  canYearScrollLeft.value = track.scrollLeft > 4
  canYearScrollRight.value = track.scrollLeft < maxScroll - 4
}

function scrollYearTrack(direction) {
  const track = yearTrackRef.value
  if (!track) {
    return
  }

  const offset = Math.max(180, Math.floor(track.clientWidth * 0.55))
  track.scrollBy({
    left: direction * offset,
    behavior: 'smooth'
  })
}

function handleYearWheel(event) {
  const track = yearTrackRef.value
  if (!track) {
    return
  }

  if (Math.abs(event.deltaY) <= Math.abs(event.deltaX)) {
    return
  }

  track.scrollLeft += event.deltaY
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
  return date.toLocaleString(tr('ru-RU', 'kk-KZ'))
}

function statusLabel(status) {
  const normalized = String(status ?? '').toLowerCase()
  if (normalized === 'completed' || normalized === 'approved') {
    return tr('Завершено', 'Аяқталған')
  }
  if (normalized === 'rejected') {
    return tr('Отклонено', 'Қабылданбады')
  }
  if (normalized === 'overdue') {
    return tr('Просрочено', 'Мерзімі өткен')
  }
  return tr('На проверке', 'Тексерісте')
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

  await nextTick()
  updateYearScrollState()
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
      ?? tr('Не удалось загрузить данные', 'Мәліметтерді жүктеу мүмкін болмады')
  } finally {
    loading.value = false
  }
}

async function selectYear(yearValue) {
  const nextYear = String(yearValue ?? '').trim()
  if (!nextYear || selectedYear.value === nextYear) {
    return
  }

  selectedYear.value = nextYear
  loading.value = true
  clearMessages()

  try {
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось загрузить данные по году', 'Жыл бойынша мәліметтерді жүктеу мүмкін болмады')
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
      ?? tr('Не удалось загрузить данные по категории', 'Категория бойынша мәліметтерді жүктеу мүмкін болмады')
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

  return tr(fallback, fallback)
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
    errorMessage.value = await extractErrorMessage(error, tr('Не удалось загрузить файл', 'Файлды жүктеу мүмкін болмады'))
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
      ? tr('Заполните формулу', 'Формуланы толтырыңыз')
      : tr('Заполните причину отклонения', 'Қабылдамау себебін толтырыңыз')
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
      successMessage.value = tr('Отчет принят', 'Отчет қабылданды')
    } else {
      await reviewPlanReport(reviewRowId.value, {
        action: 'reject',
        review_note: normalizedText
      })
      successMessage.value = tr('Отчет отклонен', 'Отчет қабылданбады')
    }

    closeReviewModal()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось обновить статус', 'Статусты жаңарту мүмкін болмады')
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
    errorMessage.value = tr('Нужно загрузить минимум один документ', 'Кемінде бір құжат жүктеу міндетті')
    return
  }

  resubmitSending.value = true
  clearMessages()

  try {
    await submitPlanIndicatorReport(resubmitRow.value.indicator_id, selectedYear.value, {
      report_text: resubmitText.value.trim(),
      files: resubmitFiles.value
    })
    successMessage.value = tr('Отчет отправлен повторно', 'Отчет қайта жіберілді')
    closeResubmitModal()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось отправить отчет повторно', 'Отчетты қайта жіберу мүмкін болмады')
  } finally {
    resubmitSending.value = false
  }
}

onMounted(() => {
  initialize()
  window.addEventListener('resize', updateYearScrollState)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateYearScrollState)
})
</script>

<template>
  <section class="execution-page">
    <PageHeader
      :title="tr('ВЫПОЛНЕНИЕ ПРОГРАММЫ РАЗВИТИЯ', 'ДАМУ БАҒДАРЛАМАСЫНЫҢ ОРЫНДАЛУЫ')"
      :subtitle="pageSubtitle"
      :eyebrow="tr('Отчеты', 'Есептер')"
    />

    <div class="panel panel-strong toolbar-panel execution-toolbar">
      <div class="execution-toolbar-top">
        <div class="execution-year-strip">
          <div class="execution-year-strip-head">
            <span class="execution-year-label">{{ tr('Год', 'Жыл') }}</span>
            <div class="execution-year-nav">
              <button
                class="btn btn-ghost year-nav-btn"
                type="button"
                :disabled="!canYearScrollLeft"
                @click="scrollYearTrack(-1)"
              >
                ←
              </button>
              <button
                class="btn btn-ghost year-nav-btn"
                type="button"
                :disabled="!canYearScrollRight"
                @click="scrollYearTrack(1)"
              >
                →
              </button>
            </div>
          </div>

          <div
            class="year-strip-shell"
            :class="{
              'has-left-fade': canYearScrollLeft,
              'has-right-fade': canYearScrollRight
            }"
          >
            <div
              ref="yearTrackRef"
              class="year-track"
              @scroll="updateYearScrollState"
              @wheel.prevent="handleYearWheel"
            >
              <button
                v-for="year in years"
                :key="`execution-year-${year}`"
                type="button"
                class="year-tab"
                :class="{ 'is-active': String(year) === selectedYear }"
                @click="selectYear(year)"
              >
                {{ year }}
              </button>
            </div>
          </div>
        </div>

        <div class="execution-summary-card execution-summary-compact">
          <span class="kicker">{{ tr('Строки', 'Жолдар') }}</span>
          <strong>{{ rows.length }}</strong>
        </div>
      </div>

      <div v-if="isProrector" class="execution-categories">
        <button
          class="btn btn-ghost execution-category-btn"
          :class="{ 'is-active': prorectorCategory === 'pending' }"
          type="button"
          @click="handleProrectorCategoryChange('pending')"
        >
          {{ tr('На проверке', 'Тексерісте') }}
        </button>
        <button
          class="btn btn-ghost execution-category-btn"
          :class="{ 'is-active': prorectorCategory === 'rejected' }"
          type="button"
          @click="handleProrectorCategoryChange('rejected')"
        >
          {{ tr('Отклонено', 'Қабылданбаған') }}
        </button>
      </div>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <section class="panel panel-strong execution-table-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ tr('Отчеты и решения', 'Есептер және шешімдер') }}</h3>
          <p class="panel-subtitle">{{ tr('Файлы, текст отчета, формулы проверки и причина отклонения видны в одном потоке.', 'Файлдар, есеп мәтіні, тексеру формуласы және қабылдамау себебі бір ағынмен көрсетіледі.') }}</p>
        </div>
        <span class="kicker">{{ selectedYear || tr('Нет года', 'Жыл жоқ') }}</span>
      </div>

      <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
      <div v-else-if="!hasYears" class="empty-state">
        {{ tr('Пока нет годов.', 'Әзірге жылдар жоқ.') }}
      </div>
      <div v-else-if="rows.length === 0" class="empty-state">
        <template v-if="isProrector">
          <template v-if="isRejectedCategory">
            {{ tr(`За ${selectedYear} год нет отклоненных отчетов.`, `${selectedYear} жылына қабылданбаған есептер жоқ.`) }}
          </template>
          <template v-else>
            {{ tr(`За ${selectedYear} год нет отчетов в категории На проверке.`, `${selectedYear} жылына На проверке категориясында отчеттар жоқ.`) }}
          </template>
        </template>
        <template v-else>
          {{ tr(`За ${selectedYear} год нет отправленных отчетов.`, `${selectedYear} жылына жіберілген отчеттар жоқ.`) }}
        </template>
      </div>

      <div
        v-else-if="isAdmin"
        class="table-wrap execution-table-wrap"
      >
        <table class="table execution-table">
          <thead>
            <tr>
              <th>№</th>
              <th>{{ tr('Целевой индикатор', 'Мақсатты индикатор') }}</th>
              <th>{{ tr('Срок исполнения', 'Орындау мерзімі') }}</th>
              <th>{{ tr('Ответственные', 'Жауаптылар') }}</th>
              <th>{{ tr('Выполнение индикатора', 'Индикатор орындалуы') }}</th>
              <th>{{ tr('Статус', 'Мәртебе') }}</th>
              <th>{{ tr('Решение', 'Шешім') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.id">
              <td class="number-cell" data-label="№">{{ index + 1 }}</td>
              <td :data-label="tr('Целевой индикатор', 'Мақсатты индикатор')">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.development_indicator) === '—' }"
                  role="button"
                  tabindex="0"
                    @click="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.development_indicator)"
                    @keyup.enter="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.development_indicator)"
                    @keyup.space.prevent="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.development_indicator)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.development_indicator) }}</span>
                </div>
                <span class="planned-value-chip">{{ formatPlannedValue(row.planned_value, row.unit) }}</span>
              </td>
              <td :data-label="tr('Срок исполнения', 'Орындау мерзімі')">{{ row.execution_deadline || '—' }}</td>
              <td class="text-pretty" :data-label="tr('Ответственные', 'Жауаптылар')">{{ row.responsible || '—' }}</td>
              <td :data-label="tr('Выполнение индикатора', 'Индикатор орындалуы')">
                <div class="report-card">
                  <div
                    class="table-text-preview text-pretty"
                    :class="{ 'is-empty': textPreview(row.report_text) === '—' }"
                    role="button"
                    tabindex="0"
                    @click="openReadModal(tr('Выполнение индикатора', 'Индикатор орындалуы'), row.report_text)"
                    @keyup.enter="openReadModal(tr('Выполнение индикатора', 'Индикатор орындалуы'), row.report_text)"
                    @keyup.space.prevent="openReadModal(tr('Выполнение индикатора', 'Индикатор орындалуы'), row.report_text)"
                  >
                    <span class="table-text-preview-content">{{ textPreview(row.report_text) }}</span>
                  </div>
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
                    <span v-if="!row.files || row.files.length === 0" class="muted">{{ tr('Нет файла', 'Файл жоқ') }}</span>
                  </div>
                  <p class="meta">
                    {{ tr('Отправил:', 'Жіберген:') }} {{ row.submitted_by_name || row.submitted_by }} • {{ formatDate(row.submitted_at) }}
                  </p>
                  <p v-if="row.reviewed_by_name" class="meta">
                    {{ tr('Проверил:', 'Тексерген:') }} {{ row.reviewed_by_name }} • {{ formatDate(row.reviewed_at) }}
                  </p>
                  <p v-if="row.approval_formula" class="formula-text text-pretty">
                    {{ tr('Формула:', 'Формула:') }} {{ row.approval_formula }}
                  </p>
                  <p v-if="row.review_note" class="reject-note text-pretty">
                    {{ tr('Причина:', 'Себеп:') }} {{ row.review_note }}
                  </p>
                </div>
              </td>
              <td :data-label="tr('Статус', 'Мәртебе')">
                <span class="status-pill" :class="`status-${row.status}`">
                  {{ statusLabel(row.status) }}
                </span>
              </td>
              <td class="actions-cell" :data-label="tr('Решение', 'Шешім')">
                <template v-if="canReviewRow(row)">
                  <button class="btn btn-primary action-btn" type="button" @click="openApproveModal(row)">
                    {{ tr('Принять', 'Қабылдау') }}
                  </button>
                  <button class="btn btn-danger action-btn" type="button" @click="openRejectModal(row)">
                    {{ tr('Отклонить', 'Қабылдамау') }}
                  </button>
                </template>
                <span v-else class="muted">{{ tr('Рассмотрено', 'Қаралған') }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-else
        class="table-wrap execution-table-wrap"
      >
        <table class="table execution-table execution-table-prorector">
          <thead>
            <tr>
              <th>№</th>
              <th>{{ tr('Целевой индикатор', 'Мақсатты индикатор') }}</th>
              <th>{{ tr('Срок исполнения', 'Орындау мерзімі') }}</th>
              <th>{{ tr('Ответственные', 'Жауаптылар') }}</th>
              <th v-if="isRejectedCategory">{{ tr('Причина отклонения', 'Қабылданбау себебі') }}</th>
              <th v-else>{{ tr('Статус', 'Мәртебе') }}</th>
              <th>{{ tr('Предыдущий отчет', 'Алдыңғы есеп') }}</th>
              <th>{{ tr('Документы', 'Құжаттар') }}</th>
              <th>{{ tr('Действие', 'Әрекет') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.id">
              <td class="number-cell" data-label="№">{{ index + 1 }}</td>
              <td :data-label="tr('Целевой индикатор', 'Мақсатты индикатор')">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.development_indicator) === '—' }"
                  role="button"
                  tabindex="0"
                    @click="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.development_indicator)"
                    @keyup.enter="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.development_indicator)"
                    @keyup.space.prevent="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.development_indicator)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.development_indicator) }}</span>
                </div>
                <span class="planned-value-chip">{{ formatPlannedValue(row.planned_value, row.unit) }}</span>
              </td>
              <td :data-label="tr('Срок исполнения', 'Орындау мерзімі')">{{ row.execution_deadline || '—' }}</td>
              <td class="text-pretty" :data-label="tr('Ответственные', 'Жауаптылар')">{{ row.responsible || '—' }}</td>
              <td v-if="isRejectedCategory" class="text-pretty" :data-label="tr('Причина отклонения', 'Қабылданбау себебі')">{{ row.review_note || '—' }}</td>
              <td v-else :data-label="tr('Статус', 'Мәртебе')">
                <span class="status-pill status-pending">{{ tr('На проверке', 'Тексерісте') }}</span>
              </td>
              <td :data-label="tr('Предыдущий отчет', 'Алдыңғы есеп')">
                <div class="report-card">
                  <div
                    class="table-text-preview text-pretty"
                    :class="{ 'is-empty': textPreview(row.report_text) === '—' }"
                    role="button"
                    tabindex="0"
                    @click="openReadModal(tr('Предыдущий отчет', 'Алдыңғы есеп'), row.report_text)"
                    @keyup.enter="openReadModal(tr('Предыдущий отчет', 'Алдыңғы есеп'), row.report_text)"
                    @keyup.space.prevent="openReadModal(tr('Предыдущий отчет', 'Алдыңғы есеп'), row.report_text)"
                  >
                    <span class="table-text-preview-content">{{ textPreview(row.report_text) }}</span>
                  </div>
                  <p class="meta">
                    {{ tr('Отправлено:', 'Жіберілген:') }} {{ formatDate(row.submitted_at) }}
                  </p>
                </div>
              </td>
              <td :data-label="tr('Документы', 'Құжаттар')">
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
                  <span v-if="!row.files || row.files.length === 0" class="muted">{{ tr('Нет файла', 'Файл жоқ') }}</span>
                </div>
              </td>
              <td :data-label="tr('Действие', 'Әрекет')">
                <button
                  v-if="isRejectedCategory"
                  class="btn btn-primary action-btn"
                  type="button"
                  @click="openResubmitModal(row)"
                >
                  {{ tr('Исправить и отправить', 'Түзетіп жіберу') }}
                </button>
                <span v-else class="muted">{{ tr('Ожидание', 'Күтілуде') }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div v-if="readModalOpen" class="modal-backdrop" @click.self="closeReadModal">
      <div class="modal-card execution-read-modal">
        <h3 class="modal-title">{{ readModalTitle }}</h3>
        <div class="execution-read-content text-pretty">
          {{ readModalText }}
        </div>
        <div class="modal-actions">
          <button class="btn btn-primary" type="button" @click="closeReadModal">
            {{ tr('Закрыть', 'Жабу') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="reviewModalOpen" class="modal-backdrop" @click.self="closeReviewModal">
      <div class="modal-card execution-modal">
        <h3 class="modal-title">
          {{ reviewMode === 'approve' ? tr('Принятие отчета', 'Есепті қабылдау') : tr('Отклонение отчета', 'Есепті қабылдамау') }}
        </h3>
        <p class="modal-subtitle">
          {{ activeReviewRow?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>

        <label class="modal-label">
          <span v-if="reviewMode === 'approve'">
            {{ tr('Формула и итоговое значение', 'Формула және қорытынды сан') }}
          </span>
          <span v-else>
            {{ tr('Причина отклонения', 'Неге қабылданбады') }}
          </span>
          <textarea
            v-model="reviewText"
            class="modal-textarea"
            rows="6"
            :placeholder="reviewMode === 'approve'
              ? tr('Например: 157/525 ППС + 11587 обучающихся *100%=1,3%', 'Мысалы: 157/525 ППС + 11587 обучающихся *100%=1,3%')
              : tr('Укажите причину отклонения...', 'Қабылданбау себебін жазыңыз...')"
          />
        </label>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeReviewModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="reviewSaving"
            @click="submitReviewDecision"
          >
            {{ reviewSaving ? tr('Сохранение...', 'Сақталуда...') : (reviewMode === 'approve' ? tr('Закрыть', 'Жабу') : tr('Отправить', 'Жіберу')) }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="resubmitModalOpen" class="modal-backdrop" @click.self="closeResubmitModal">
      <div class="modal-card execution-modal">
        <h3 class="modal-title">{{ tr('Повторная отправка отклоненного отчета', 'Қабылданбаған есепті қайта жіберу') }}</h3>
        <p class="modal-subtitle">
          {{ resubmitRow?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>

        <label class="modal-label">
          {{ tr('Текст отчета', 'Есеп мәтіні') }}
          <textarea
            v-model="resubmitText"
            class="modal-textarea"
            rows="6"
            :placeholder="tr('Введите исправленный отчет...', 'Түзетілген отчетты жазыңыз...')"
          />
        </label>

        <label class="modal-label">
          {{ tr('Документы (минимум 1 файл)', 'Құжаттар (кемінде 1 файл)') }}
          <input
            class="file-input"
            type="file"
            accept=".doc,.docx,.xls,.xlsx,.ppt,.pptx,.pdf"
            multiple
            @change="handleResubmitFiles"
          />
        </label>
        <p v-if="resubmitFiles.length > 0" class="file-info">
          {{ tr('Выбранные файлы:', 'Таңдалған файлдар:') }} {{ resubmitFiles.map((file) => file.name).join(', ') }}
        </p>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeResubmitModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="resubmitSending"
            @click="submitResubmittedReport"
          >
            {{ resubmitSending ? tr('Отправка...', 'Жіберілуде...') : tr('Отправить', 'Жіберу') }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.execution-toolbar {
  display: grid;
  gap: 0.78rem;
  align-items: stretch;
}

.execution-toolbar-top {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 0.95rem;
}

.execution-year-strip {
  display: grid;
  gap: 0.45rem;
  min-width: 0;
  width: min(100%, 940px);
}

.execution-year-strip-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
}

.execution-year-label {
  color: var(--muted);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.execution-year-nav {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.year-nav-btn {
  min-height: 2rem;
  padding: 0.35rem 0.62rem;
}

.year-strip-shell {
  position: relative;
  overflow: hidden;
  border-radius: 16px;
  border: 1px solid rgba(16, 33, 42, 0.1);
  background: rgba(255, 255, 255, 0.86);
  max-width: 100%;
}

.year-strip-shell::before,
.year-strip-shell::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  width: 42px;
  pointer-events: none;
  z-index: 2;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.year-strip-shell::before {
  left: 0;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.98), rgba(255, 255, 255, 0));
}

.year-strip-shell::after {
  right: 0;
  background: linear-gradient(270deg, rgba(255, 255, 255, 0.98), rgba(255, 255, 255, 0));
}

.year-strip-shell.has-left-fade::before {
  opacity: 1;
}

.year-strip-shell.has-right-fade::after {
  opacity: 1;
}

.year-track {
  display: flex;
  align-items: center;
  gap: 0.38rem;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-behavior: smooth;
  padding: 0.3rem;
  scrollbar-width: none;
}

.year-track::-webkit-scrollbar {
  display: none;
}

.year-tab {
  border: 1px solid rgba(16, 33, 42, 0.12);
  background: rgba(255, 255, 255, 0.96);
  color: #324754;
  min-height: 1.88rem;
  min-width: 3.4rem;
  padding: 0.28rem 0.68rem;
  border-radius: 999px;
  font-size: 0.82rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}

.year-tab.is-active {
  border-color: rgba(17, 120, 111, 0.5);
  background: linear-gradient(135deg, rgba(17, 120, 111, 0.2), rgba(17, 120, 111, 0.1));
  color: #0f5e57;
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

.execution-summary-compact {
  min-width: 90px;
  justify-items: end;
  text-align: right;
}

.execution-table-card {
  padding: 1.2rem;
  min-width: 0;
  overflow: hidden;
}

.execution-table-wrap {
  max-width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
}

.execution-table {
  min-width: 1260px;
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

.planned-value-chip {
  display: inline-flex;
  align-items: center;
  margin-top: 0.35rem;
  border: 1px solid #d8e0ea;
  border-radius: 999px;
  padding: 0.2rem 0.58rem;
  color: #475569;
  background: #f8fafc;
  font-size: 0.78rem;
  font-weight: 700;
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

.execution-read-modal {
  width: min(760px, 100%);
}

.execution-read-content {
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
    gap: 0.7rem;
  }

  .execution-toolbar-top {
    grid-template-columns: 1fr;
    align-items: stretch;
    gap: 0.72rem;
  }

  .execution-year-strip {
    width: 100%;
  }

  .execution-categories {
    width: 100%;
  }

  .execution-summary-compact {
    justify-items: start;
    text-align: left;
  }
}

@media (max-width: 980px) {
  .execution-table-card {
    padding: 0.95rem;
  }

  .table-wrap {
    overflow: visible;
    border: 0;
    box-shadow: none;
    background: transparent;
  }

  .execution-table {
    min-width: 0;
    display: block;
  }

  .execution-table thead {
    display: none;
  }

  .execution-table tbody {
    display: grid;
    gap: 0.82rem;
  }

  .execution-table tbody tr {
    display: block;
    padding: 0.72rem 0.86rem;
    border: 1px solid var(--border);
    border-radius: 20px;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: var(--shadow-soft);
  }

  .execution-table tbody td {
    display: grid;
    grid-template-columns: minmax(130px, 38%) 1fr;
    gap: 0.56rem;
    padding: 0.5rem 0.1rem;
    border-bottom: 1px dashed rgba(16, 33, 42, 0.12);
  }

  .execution-table tbody td:last-child {
    border-bottom: 0;
  }

  .execution-table tbody td::before {
    content: attr(data-label);
    color: var(--muted);
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .number-cell {
    width: auto;
    text-align: left;
  }

  .actions-cell {
    min-width: 0;
  }

  .year-track {
    padding: 0.32rem;
  }
}

@media (max-width: 640px) {
  .execution-table tbody td {
    grid-template-columns: 1fr;
    gap: 0.4rem;
  }

  .execution-table tbody td::before {
    font-size: 0.68rem;
  }
}
</style>
