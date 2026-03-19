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
const adminCategory = ref('pending')
const prorectorCategory = ref('pending')
const yearTrackRef = ref(null)
const canYearScrollLeft = ref(false)
const canYearScrollRight = ref(false)
const readModalOpen = ref(false)
const readModalTitle = ref('')
const readModalText = ref('')
const reportDetailsModalOpen = ref(false)
const reportDetailsTitle = ref('')
const reportDetailsRow = ref(null)

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
const isAdminPendingCategory = computed(() => isAdmin.value && adminCategory.value === 'pending')
const isAdminCompletedCategory = computed(() => isAdmin.value && adminCategory.value === 'completed')
const isAdminRejectedCategory = computed(() => isAdmin.value && adminCategory.value === 'rejected')
const isProrectorCompletedCategory = computed(() => isProrector.value && prorectorCategory.value === 'completed')
const isProrectorRejectedCategory = computed(() => isProrector.value && prorectorCategory.value === 'rejected')
const hasYears = computed(() => years.value.length > 0)
const canLoadYear = computed(() => selectedYear.value !== '')
const activeReviewRow = computed(() => rows.value.find((item) => item.id === reviewRowId.value) ?? null)
const pageSubtitle = computed(() => {
  if (isAdmin.value) {
    if (isAdminCompletedCategory.value) {
      return tr('Утвержденные отчеты с итоговой формулой и подтверждающими файлами', 'Қорытынды формуласы мен файлдары бар бекітілген есептер')
    }
    if (isAdminRejectedCategory.value) {
      return tr('Отклоненные отчеты с причинами отклонения', 'Қабылданбаған есептер және қабылданбау себептері')
    }
    return tr('Индикаторы, ожидающие проверки и решения администратора', 'Әкімші тексеруін және шешімін күтіп тұрған индикаторлар')
  }
  if (isProrector.value) {
    if (isProrectorRejectedCategory.value) {
      return tr('Отклоненные отчеты. Можно исправить и отправить повторно', 'Қабылданбаған есептер. Түзетіп, қайта жіберуге болады')
    }
    if (isProrectorCompletedCategory.value) {
      return tr('Принятые отчеты с итоговой формулой администратора', 'Әкімші қабылдаған есептер және қорытынды формула')
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

function getFileCount(row) {
  return Array.isArray(row?.files) ? row.files.length : 0
}

function openReportDetails(row, title) {
  if (!row) {
    return
  }

  reportDetailsRow.value = row
  reportDetailsTitle.value = title
  reportDetailsModalOpen.value = true
}

function closeReportDetails() {
  reportDetailsModalOpen.value = false
  reportDetailsTitle.value = ''
  reportDetailsRow.value = null
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

function canReviewRow(row) {
  const normalized = String(row?.status ?? '').toLowerCase()
  return normalized === 'pending'
}

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
    await nextTick()
    updateYearScrollState()
    return
  }

  const currentSelection = Number(selectedYear.value)
  const preferredYear = normalizedYears.includes(currentYear)
    ? currentYear
    : normalizedYears[normalizedYears.length - 1]

  if (!selectedYear.value || !normalizedYears.includes(currentSelection)) {
    selectedYear.value = String(preferredYear)
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
    options.status = prorectorCategory.value
  } else if (isAdmin.value) {
    options.status = adminCategory.value
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

async function handleAdminCategoryChange(category) {
  if (!isAdmin.value || adminCategory.value === category) {
    return
  }

  adminCategory.value = category
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

      <div v-if="isAdmin" class="execution-categories">
        <button
          class="btn btn-ghost execution-category-btn"
          :class="{ 'is-active': adminCategory === 'pending' }"
          type="button"
          @click="handleAdminCategoryChange('pending')"
        >
          {{ tr('На проверке', 'Тексерісте') }}
        </button>
        <button
          class="btn btn-ghost execution-category-btn"
          :class="{ 'is-active': adminCategory === 'completed' }"
          type="button"
          @click="handleAdminCategoryChange('completed')"
        >
          {{ tr('Принято', 'Қабылданды') }}
        </button>
        <button
          class="btn btn-ghost execution-category-btn"
          :class="{ 'is-active': adminCategory === 'rejected' }"
          type="button"
          @click="handleAdminCategoryChange('rejected')"
        >
          {{ tr('Отклонено', 'Қабылданбаған') }}
        </button>
      </div>

      <div v-else-if="isProrector" class="execution-categories">
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
          :class="{ 'is-active': prorectorCategory === 'completed' }"
          type="button"
          @click="handleProrectorCategoryChange('completed')"
        >
          {{ tr('Принято', 'Қабылданды') }}
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
          <template v-if="isProrectorRejectedCategory">
            {{ tr(`За ${selectedYear} год нет отклоненных отчетов.`, `${selectedYear} жылына қабылданбаған есептер жоқ.`) }}
          </template>
          <template v-else-if="isProrectorCompletedCategory">
            {{ tr(`За ${selectedYear} год нет принятых отчетов.`, `${selectedYear} жылына қабылданған есептер жоқ.`) }}
          </template>
          <template v-else>
            {{ tr(`За ${selectedYear} год нет отчетов в категории На проверке.`, `${selectedYear} жылына На проверке категориясында отчеттар жоқ.`) }}
          </template>
        </template>
        <template v-else-if="isAdmin">
          <template v-if="isAdminCompletedCategory">
            {{ tr(`За ${selectedYear} год нет принятых отчетов.`, `${selectedYear} жылына қабылданған есептер жоқ.`) }}
          </template>
          <template v-else-if="isAdminRejectedCategory">
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
              <th v-if="isAdminCompletedCategory">{{ tr('Итог', 'Қорытынды') }}</th>
              <th v-else-if="isAdminRejectedCategory">{{ tr('Причина отклонения', 'Қабылданбау себебі') }}</th>
              <th v-if="isAdminPendingCategory">{{ tr('Решение', 'Шешім') }}</th>
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
              <td :data-label="tr('Ответственные', 'Жауаптылар')">
                <div
                  class="table-text-preview text-pretty responsible-preview"
                  :class="{ 'is-empty': textPreview(row.responsible) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal(tr('Ответственные', 'Жауаптылар'), row.responsible)"
                  @keyup.enter="openReadModal(tr('Ответственные', 'Жауаптылар'), row.responsible)"
                  @keyup.space.prevent="openReadModal(tr('Ответственные', 'Жауаптылар'), row.responsible)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.responsible) }}</span>
                </div>
              </td>
              <td :data-label="tr('Выполнение индикатора', 'Индикатор орындалуы')">
                <div class="report-compact">
                  <div
                    class="table-text-preview text-pretty report-compact-preview"
                    :class="{ 'is-empty': textPreview(row.report_text) === '—' }"
                    role="button"
                    tabindex="0"
                    @click="openReportDetails(row, tr('Выполнение индикатора', 'Индикатор орындалуы'))"
                    @keyup.enter="openReportDetails(row, tr('Выполнение индикатора', 'Индикатор орындалуы'))"
                    @keyup.space.prevent="openReportDetails(row, tr('Выполнение индикатора', 'Индикатор орындалуы'))"
                  >
                    <span class="table-text-preview-content">{{ textPreview(row.report_text) }}</span>
                  </div>
                  <div class="report-compact-footer">
                    <span class="planned-value-chip report-count-chip">
                      {{ tr('Файлы', 'Файлдар') }}: {{ getFileCount(row) }}
                    </span>
                    <button
                      class="btn btn-ghost report-details-btn"
                      type="button"
                      @click="openReportDetails(row, tr('Выполнение индикатора', 'Индикатор орындалуы'))"
                    >
                      {{ tr('Подробнее', 'Толығырақ') }}
                    </button>
                  </div>
                  <p class="meta">
                    {{ tr('Отправил:', 'Жіберген:') }} {{ row.submitted_by_name || row.submitted_by }} • {{ formatDate(row.submitted_at) }}
                  </p>
                  <p v-if="row.reviewed_by_name" class="meta">
                    {{ tr('Проверил:', 'Тексерген:') }} {{ row.reviewed_by_name }} • {{ formatDate(row.reviewed_at) }}
                  </p>
                </div>
              </td>
              <td v-if="isAdminCompletedCategory" :data-label="tr('Итог', 'Қорытынды')">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.approval_formula) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal(tr('Итог', 'Қорытынды'), row.approval_formula)"
                  @keyup.enter="openReadModal(tr('Итог', 'Қорытынды'), row.approval_formula)"
                  @keyup.space.prevent="openReadModal(tr('Итог', 'Қорытынды'), row.approval_formula)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.approval_formula) }}</span>
                </div>
              </td>
              <td v-else-if="isAdminRejectedCategory" :data-label="tr('Причина отклонения', 'Қабылданбау себебі')">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.review_note) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal(tr('Причина отклонения', 'Қабылданбау себебі'), row.review_note)"
                  @keyup.enter="openReadModal(tr('Причина отклонения', 'Қабылданбау себебі'), row.review_note)"
                  @keyup.space.prevent="openReadModal(tr('Причина отклонения', 'Қабылданбау себебі'), row.review_note)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.review_note) }}</span>
                </div>
              </td>
              <td v-if="isAdminPendingCategory" class="actions-cell" :data-label="tr('Решение', 'Шешім')">
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
              <th v-if="isProrectorCompletedCategory">{{ tr('Итог', 'Қорытынды') }}</th>
              <th v-else-if="isProrectorRejectedCategory">{{ tr('Причина отклонения', 'Қабылданбау себебі') }}</th>
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
              <td :data-label="tr('Ответственные', 'Жауаптылар')">
                <div
                  class="table-text-preview text-pretty responsible-preview"
                  :class="{ 'is-empty': textPreview(row.responsible) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal(tr('Ответственные', 'Жауаптылар'), row.responsible)"
                  @keyup.enter="openReadModal(tr('Ответственные', 'Жауаптылар'), row.responsible)"
                  @keyup.space.prevent="openReadModal(tr('Ответственные', 'Жауаптылар'), row.responsible)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.responsible) }}</span>
                </div>
              </td>
              <td v-if="isProrectorCompletedCategory" :data-label="tr('Итог', 'Қорытынды')">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.approval_formula) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal(tr('Итог', 'Қорытынды'), row.approval_formula)"
                  @keyup.enter="openReadModal(tr('Итог', 'Қорытынды'), row.approval_formula)"
                  @keyup.space.prevent="openReadModal(tr('Итог', 'Қорытынды'), row.approval_formula)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.approval_formula) }}</span>
                </div>
              </td>
              <td v-else-if="isProrectorRejectedCategory" :data-label="tr('Причина отклонения', 'Қабылданбау себебі')">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.review_note) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal(tr('Причина отклонения', 'Қабылданбау себебі'), row.review_note)"
                  @keyup.enter="openReadModal(tr('Причина отклонения', 'Қабылданбау себебі'), row.review_note)"
                  @keyup.space.prevent="openReadModal(tr('Причина отклонения', 'Қабылданбау себебі'), row.review_note)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.review_note) }}</span>
                </div>
              </td>
              <td :data-label="tr('Предыдущий отчет', 'Алдыңғы есеп')">
                <div class="report-compact">
                  <div
                    class="table-text-preview text-pretty report-compact-preview"
                    :class="{ 'is-empty': textPreview(row.report_text) === '—' }"
                    role="button"
                    tabindex="0"
                    @click="openReportDetails(row, tr('Предыдущий отчет', 'Алдыңғы есеп'))"
                    @keyup.enter="openReportDetails(row, tr('Предыдущий отчет', 'Алдыңғы есеп'))"
                    @keyup.space.prevent="openReportDetails(row, tr('Предыдущий отчет', 'Алдыңғы есеп'))"
                  >
                    <span class="table-text-preview-content">{{ textPreview(row.report_text) }}</span>
                  </div>
                  <div class="report-compact-footer">
                    <span class="planned-value-chip report-count-chip">
                      {{ tr('Файлы', 'Файлдар') }}: {{ getFileCount(row) }}
                    </span>
                    <button
                      class="btn btn-ghost report-details-btn"
                      type="button"
                      @click="openReportDetails(row, tr('Предыдущий отчет', 'Алдыңғы есеп'))"
                    >
                      {{ tr('Подробнее', 'Толығырақ') }}
                    </button>
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
                  v-if="isProrectorRejectedCategory"
                  class="btn btn-primary action-btn"
                  type="button"
                  @click="openResubmitModal(row)"
                >
                  {{ tr('Исправить и отправить', 'Түзетіп жіберу') }}
                </button>
                <span v-else-if="isProrectorCompletedCategory" class="muted">{{ tr('Рассмотрено', 'Қаралған') }}</span>
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

    <div v-if="reportDetailsModalOpen" class="modal-backdrop" @click.self="closeReportDetails">
      <div class="modal-card execution-report-modal">
        <h3 class="modal-title">{{ reportDetailsTitle }}</h3>
        <p class="modal-subtitle text-pretty">
          {{ reportDetailsRow?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <div class="report-details-grid">
          <section class="report-details-section">
            <h4 class="report-details-title">{{ tr('Текст отчета', 'Есеп мәтіні') }}</h4>
            <div class="execution-read-content text-pretty report-details-content">
              {{ textPreview(reportDetailsRow?.report_text) }}
            </div>
          </section>

          <section class="report-details-section">
            <h4 class="report-details-title">{{ tr('Документы', 'Құжаттар') }}</h4>
            <div class="files-list report-details-files">
              <button
                v-for="file in (reportDetailsRow?.files ?? [])"
                :key="file.id"
                class="btn btn-ghost report-file-chip"
                type="button"
                @click="handleDownload(file)"
              >
                {{ file.file_name }}
              </button>
              <span
                v-if="!reportDetailsRow?.files || reportDetailsRow.files.length === 0"
                class="muted"
              >
                {{ tr('Нет файла', 'Файл жоқ') }}
              </span>
            </div>
          </section>
        </div>

        <div class="report-details-meta">
          <p class="meta">
            {{ tr('Отправил:', 'Жіберген:') }} {{ reportDetailsRow?.submitted_by_name || reportDetailsRow?.submitted_by || '—' }} • {{ formatDate(reportDetailsRow?.submitted_at) }}
          </p>
          <p v-if="reportDetailsRow?.reviewed_by_name" class="meta">
            {{ tr('Проверил:', 'Тексерген:') }} {{ reportDetailsRow.reviewed_by_name }} • {{ formatDate(reportDetailsRow.reviewed_at) }}
          </p>
          <p v-if="reportDetailsRow?.approval_formula" class="formula-text text-pretty">
            {{ tr('Формула:', 'Формула:') }} {{ reportDetailsRow.approval_formula }}
          </p>
          <p v-if="reportDetailsRow?.review_note" class="reject-note text-pretty">
            {{ tr('Причина:', 'Себеп:') }} {{ reportDetailsRow.review_note }}
          </p>
        </div>

        <div class="modal-actions">
          <button class="btn btn-primary" type="button" @click="closeReportDetails">
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
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <div
          v-if="reviewMode === 'approve' && activeReviewRow?.evaluation_formula"
          class="execution-formula-hint"
        >
          <span class="execution-formula-hint-label">
            {{ tr('Формула оценки индикатора', 'Индикаторды бағалау формуласы') }}
          </span>
          <div class="execution-formula-hint-body text-pretty">
            {{ activeReviewRow.evaluation_formula }}
          </div>
        </div>

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
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

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
  gap: 0.5rem;
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
  min-height: 2.28rem;
  min-width: 2.28rem;
  padding: 0.35rem 0.62rem;
  border-radius: 999px;
  border-color: rgba(221, 224, 242, 0.92);
  background: rgba(248, 249, 255, 0.82);
  box-shadow: 0 10px 24px rgba(112, 123, 184, 0.08);
}

.year-strip-shell {
  position: relative;
  overflow: hidden;
  border-radius: 999px;
  border: 1px solid rgba(221, 224, 242, 0.9);
  background: rgba(248, 249, 255, 0.92);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.86),
    0 10px 28px rgba(112, 123, 184, 0.08);
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
  background: linear-gradient(90deg, rgba(248, 249, 255, 0.98), rgba(248, 249, 255, 0));
}

.year-strip-shell::after {
  right: 0;
  background: linear-gradient(270deg, rgba(248, 249, 255, 0.98), rgba(248, 249, 255, 0));
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
  gap: 0.5rem;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-behavior: smooth;
  padding: 0.25rem;
  scrollbar-width: none;
}

.year-track::-webkit-scrollbar {
  display: none;
}

.year-tab {
  border: 1px solid transparent;
  background: transparent;
  color: var(--muted-strong);
  min-height: 2.7rem;
  min-width: 3.55rem;
  padding: 0.72rem 1rem;
  border-radius: 999px;
  font-size: 0.94rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color 0.22s ease, color 0.22s ease, transform 0.22s ease, box-shadow 0.22s ease;
}

.year-tab:hover {
  color: var(--text);
  background: rgba(255, 255, 255, 0.5);
}

.year-tab.is-active {
  border-color: rgba(255, 255, 255, 0.75);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.74), rgba(245, 248, 255, 0.42)),
    linear-gradient(135deg, rgba(154, 188, 255, 0.16), rgba(255, 255, 255, 0.1) 40%, rgba(139, 234, 220, 0.12));
  color: var(--text);
  box-shadow:
    0 16px 34px rgba(110, 120, 180, 0.16),
    0 6px 14px rgba(255, 255, 255, 0.22),
    inset 0 1px 0 rgba(255, 255, 255, 0.88),
    inset 0 -1px 0 rgba(198, 208, 245, 0.32);
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

.responsible-preview .table-text-preview-content {
  -webkit-line-clamp: 3;
  -webkit-mask-image: linear-gradient(180deg, #000 76%, transparent);
  mask-image: linear-gradient(180deg, #000 76%, transparent);
}

.report-compact {
  display: grid;
  gap: 0.46rem;
  align-content: start;
}

.report-compact-preview .table-text-preview-content {
  -webkit-line-clamp: 3;
  -webkit-mask-image: linear-gradient(180deg, #000 74%, transparent);
  mask-image: linear-gradient(180deg, #000 74%, transparent);
}

.report-compact-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.45rem;
  flex-wrap: wrap;
}

.report-count-chip {
  margin-top: 0;
}

.report-details-btn {
  min-height: auto;
  padding: 0.36rem 0.72rem;
  font-size: 0.78rem;
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

.execution-report-modal {
  width: min(860px, 100%);
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

.execution-formula-hint {
  display: grid;
  gap: 0.45rem;
  margin-top: 0.4rem;
  padding: 0.85rem 0.95rem;
  border: 1px solid rgba(17, 120, 111, 0.16);
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(239, 249, 247, 0.9), rgba(255, 255, 255, 0.94));
}

.execution-formula-hint-label {
  color: var(--muted-strong);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.execution-formula-hint-body {
  color: var(--text);
  line-height: 1.6;
  white-space: pre-wrap;
}

.report-details-grid {
  display: grid;
  gap: 0.78rem;
  margin-top: 0.7rem;
}

.report-details-title {
  margin: 0;
  font-size: 0.84rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--muted);
}

.report-details-content {
  margin-top: 0.45rem;
}

.report-details-files {
  margin-top: 0.45rem;
  max-height: min(24vh, 180px);
  overflow: auto;
}

.report-details-meta {
  display: grid;
  gap: 0.45rem;
  margin-top: 0.82rem;
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

  .report-details-btn {
    width: 100%;
    justify-content: center;
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
