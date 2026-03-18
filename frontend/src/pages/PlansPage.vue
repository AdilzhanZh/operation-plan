<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import {
  downloadPlanReportFile,
  fetchPlanIndicators,
  fetchPlanReports,
  reviewPlanReport,
  savePlanIndicator,
  submitPlanIndicatorReport,
  updatePlanReport
} from '../services/plan.service'
import { fetchProrectorsRequest } from '../services/user.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()
const { tr } = useLocale()

const directionOptions = [
  {
    value: 'Академическое превосходство и интернационализация образования',
    ru: 'Академическое превосходство и интернационализация образования',
    kz: 'Академиялық озықтық және білім беруді интернационалдандыру'
  },
  {
    value: 'РАЗВИТИЕ НАУКИ И МЕЖДУНАРОДНОГО СОТРУДНИЧЕСТВА',
    ru: 'РАЗВИТИЕ НАУКИ И МЕЖДУНАРОДНОГО СОТРУДНИЧЕСТВА',
    kz: 'Ғылым мен халықаралық ынтымақтастықты дамыту'
  },
  {
    value: 'ЦИФРОВИЗАЦИЯ И МОДЕРНИЗАЦИЯ ИНФРАСТРУКТУРЫ',
    ru: 'ЦИФРОВИЗАЦИЯ И МОДЕРНИЗАЦИЯ ИНФРАСТРУКТУРЫ',
    kz: 'Инфрақұрылымды цифрландыру және жаңғырту'
  }
]

const selectedYear = ref(String(new Date().getFullYear()))
const rows = ref([])
const prorectors = ref([])
const loading = ref(false)
const savingIndicatorId = ref(null)
const errorMessage = ref('')
const successMessage = ref('')
const selectedResponsibleFilter = ref('')
const selectedDirectionFilter = ref('')
const reportFilterModalOpen = ref(false)
const reportExportMenuOpen = ref(false)
const activeViewTab = ref('plans')
const tabTransitionName = ref('plans-slide-left')
const reportModalOpen = ref(false)
const reportIndicatorId = ref(null)
const reportText = ref('')
const reportFiles = ref([])
const reportSending = ref(false)
const reportsHistoryModalOpen = ref(false)
const reportsHistoryIndicator = ref(null)
const reportsHistoryItems = ref([])
const reportsHistoryLoading = ref(false)
const historyEditModalOpen = ref(false)
const historyEditReportId = ref(null)
const historyEditText = ref('')
const historyEditSaving = ref(false)
const historyReviewModalOpen = ref(false)
const historyReviewMode = ref('approve')
const historyReviewReportId = ref(null)
const historyReviewText = ref('')
const historyReviewSaving = ref(false)
const readModalOpen = ref(false)
const readModalTitle = ref('')
const readModalText = ref('')
const rowEditModalOpen = ref(false)
const rowEditIndicatorId = ref(null)
const rowEditForm = ref({
  execution_start_date: '',
  execution_end_date: '',
  responsible_user_ids: []
})
const nowTimestamp = ref(Date.now())
let countdownIntervalID = null

const isAdmin = computed(() => authStore.user?.role === 'admin')
const isProrector = computed(() => authStore.user?.role === 'prorector')
const canUseReportsTab = computed(() => isAdmin.value || isProrector.value)
const canLoadYear = computed(() => selectedYear.value !== '')
const activeReportRow = computed(() => rows.value.find((item) => item.indicator_id === reportIndicatorId.value) ?? null)
const activeHistoryReport = computed(() => reportsHistoryItems.value.find((item) => item.id === historyEditReportId.value) ?? null)
const activeHistoryReviewReport = computed(() => reportsHistoryItems.value.find((item) => item.id === historyReviewReportId.value) ?? null)
const reportFilters = ref(createDefaultReportFilters())
const reportFilterDraft = ref(createDefaultReportFilters())
const reportStatusOptions = [
  { value: 'pending', ru: 'На проверке', kz: 'Тексерісте' },
  { value: 'completed', ru: 'Принято', kz: 'Қабылданды' },
  { value: 'rejected', ru: 'Отклонено', kz: 'Қабылданбаған' },
  { value: 'without_report', ru: 'Без отчета', kz: 'Есеп жоқ' },
  { value: 'in_progress', ru: 'В работе', kz: 'Жұмыста' },
  { value: 'overdue', ru: 'Просрочено', kz: 'Мерзімі өткен' },
  { value: 'upcoming', ru: 'Срок еще не наступил', kz: 'Уақыты әлі келген жоқ' }
]
const activePanelSubtitle = computed(() => {
  if (activeViewTab.value === 'reports') {
    return tr(
      'Выберите индикатор и просмотрите все отправленные отчеты. Для проректора доступна повторная отправка.',
      'Индикаторды таңдап, жіберілген есептердің толық тарихын көріңіз. Проректор үшін қайта жіберу қолжетімді.'
    )
  }
  return tr(
    'Редактирование доступно администратору, отправка отчета - назначенному проректору.',
    'Өңдеу әкімшіге қолжетімді, есеп жіберу бекітілген проректорға арналған.'
  )
})
const visibleRows = computed(() => {
  let filtered = rows.value

  const direction = String(selectedDirectionFilter.value ?? '').trim()
  if (direction) {
    filtered = filtered.filter((row) => String(row.direction ?? '').trim() === direction)
  }

  if (!isAdmin.value) {
    return filtered
  }

  const responsibleID = Number(selectedResponsibleFilter.value)
  if (!Number.isInteger(responsibleID) || responsibleID <= 0) {
    return filtered
  }

  return filtered.filter((row) => (row.responsible_user_ids ?? []).includes(responsibleID))
})
const hasRows = computed(() => rows.value.length > 0)
const hasVisibleRows = computed(() => visibleRows.value.length > 0)
const reportBaseRows = computed(() => rows.value.filter((row) => String(row?.schedule_status ?? '').trim().toLowerCase() !== 'not_filled'))
const hasReportBaseRows = computed(() => reportBaseRows.value.length > 0)
const hasVisibleReportRows = computed(() => visibleReportRows.value.length > 0)
const allReportStatusValues = computed(() => reportStatusOptions.map((option) => option.value))
const reportStatusSelectionCount = computed(() => {
  const selected = new Set((reportFilterDraft.value.statuses ?? []).map((value) => String(value)))
  return allReportStatusValues.value.filter((value) => selected.has(value)).length
})
const allReportResponsibleIds = computed(() => prorectors.value.map((item) => String(item.id)))
const reportResponsibleSelectionCount = computed(() => {
  const selected = new Set((reportFilterDraft.value.responsible_ids ?? []).map((value) => String(value)))
  return allReportResponsibleIds.value.filter((id) => selected.has(id)).length
})
const visibleReportRows = computed(() => {
  let filtered = reportBaseRows.value
  const filters = reportFilters.value
  const selectedStatuses = new Set((filters.statuses ?? []).map((value) => String(value).trim()).filter(Boolean))
  const selectedDirections = new Set((filters.directions ?? []).map((value) => String(value).trim()).filter(Boolean))
  const selectedResponsibleIDs = new Set((filters.responsible_ids ?? []).map((value) => String(value).trim()).filter(Boolean))
  const periodFrom = parseISODate(filters.period_from, false)
  const periodTo = parseISODate(filters.period_to, true)

  const includeWithoutReport = selectedStatuses.has('without_report')

  if (selectedStatuses.size === 0) {
    filtered = filtered.filter((row) => hasIndicatorReport(row))
  } else {
    filtered = filtered.filter((row) => {
      if (!includeWithoutReport && !hasIndicatorReport(row)) {
        return false
      }
      for (const status of selectedStatuses) {
        if (matchesReportStatusFilter(row, status)) {
          return true
        }
      }
      return false
    })
  }

  if (selectedDirections.size > 0) {
    filtered = filtered.filter((row) => selectedDirections.has(String(row.direction ?? '').trim()))
  }

  if (selectedResponsibleIDs.size > 0) {
    filtered = filtered.filter((row) => {
      const ids = Array.isArray(row.responsible_user_ids) ? row.responsible_user_ids : []
      return ids.some((id) => selectedResponsibleIDs.has(String(id)))
    })
  }

  if (periodFrom || periodTo) {
    filtered = filtered.filter((row) => matchesReportPeriodFilter(row, periodFrom, periodTo))
  }

  return filtered
})
const activeVisibleCount = computed(() => (activeViewTab.value === 'reports' ? visibleReportRows.value.length : visibleRows.value.length))
const activeRowEdit = computed(() => rows.value.find((item) => item.indicator_id === rowEditIndicatorId.value) ?? null)

function createDefaultReportFilters() {
  return {
    statuses: [],
    responsible_ids: [],
    directions: [],
    period_from: '',
    period_to: ''
  }
}

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

  const response = await fetchPlanIndicators(selectedYear.value, {
    include_submitted: isProrector.value
  })
  rows.value = (response.items ?? []).map((item) => ({
    ...item,
    direction: item.direction ?? '',
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

function setActiveViewTab(tab) {
  if (tab === 'reports' && !canUseReportsTab.value) {
    return
  }
  if (activeViewTab.value === tab) {
    return
  }
  const currentIndex = activeViewTab.value === 'reports' ? 1 : 0
  const nextIndex = tab === 'reports' ? 1 : 0
  tabTransitionName.value = nextIndex > currentIndex ? 'plans-slide-left' : 'plans-slide-right'
  reportExportMenuOpen.value = false
  activeViewTab.value = tab
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

function matchesReportStatusFilter(row, status) {
  const reportStatus = String(row?.report_status ?? '').trim().toLowerCase()
  const scheduleStatus = String(row?.schedule_status ?? '').trim().toLowerCase()

  switch (status) {
    case 'pending':
    case 'completed':
    case 'rejected':
      return reportStatus === status
    case 'without_report':
      return reportStatus === ''
    case 'overdue':
    case 'in_progress':
    case 'upcoming':
    case 'not_filled':
      return scheduleStatus === status
    default:
      return true
  }
}

function hasIndicatorReport(row) {
  return String(row?.report_status ?? '').trim() !== ''
}

function matchesReportPeriodFilter(row, periodFrom, periodTo) {
  const startDate = parseISODate(row?.execution_start_date, false)
  const endDate = parseISODate(row?.execution_end_date, true)

  if (!startDate || !endDate) {
    return false
  }
  if (periodFrom && endDate < periodFrom) {
    return false
  }
  if (periodTo && startDate > periodTo) {
    return false
  }
  return true
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
    return tr('Не заполнено полностью', 'Толық толтырылмаған')
  }
  if (normalized === 'in_progress') {
    return tr('Время идет', 'Уақыт өтіп жатыр')
  }
  if (normalized === 'overdue') {
    return tr('Срок истек', 'Уақыты өтіп кетті')
  }
  if (normalized === 'upcoming') {
    return tr('Срок еще не наступил', 'Уақыты әлі келген жоқ')
  }
  return tr('Срок не задан', 'Мерзім қойылмаған')
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
      ?? tr('Ошибка загрузки раздела Plans', 'Plans бөлімін жүктеу кезінде қате болды')
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
      errorMessage.value = tr('Для срока исполнения обязательны даты начала и окончания', 'Срок исполнения үшін басталу және аяқталу күні міндетті')
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
    successMessage.value = tr(`Индикатор №${row.indicator_id} сохранен`, `Индикатор №${row.indicator_id} сақталды`)
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Ошибка при сохранении', 'Сақтау кезінде қате болды')
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

function reportStatusLabel(status) {
  const normalized = String(status ?? '').toLowerCase()
  if (normalized === 'completed') {
    return tr('Принято', 'Қабылданды')
  }
  if (normalized === 'rejected') {
    return tr('Отклонено', 'Қабылданбаған')
  }
  return tr('На проверке', 'Тексерісте')
}

function reportStatusClass(status) {
  const normalized = String(status ?? '').toLowerCase()
  if (normalized === 'completed') {
    return 'report-status-completed'
  }
  if (normalized === 'rejected') {
    return 'report-status-rejected'
  }
  return 'report-status-pending'
}

function reportFilterStatusLabel(status) {
  switch (status) {
    case 'pending':
      return tr('На проверке', 'Тексерісте')
    case 'completed':
      return tr('Принято', 'Қабылданды')
    case 'rejected':
      return tr('Отклонено', 'Қабылданбаған')
    case 'without_report':
      return tr('Без отчета', 'Есеп жоқ')
    case 'overdue':
      return tr('Просрочено', 'Мерзімі өткен')
    case 'in_progress':
      return tr('В работе', 'Жұмыста')
    case 'upcoming':
      return tr('Срок еще не наступил', 'Уақыты әлі келген жоқ')
    case 'not_filled':
      return tr('Не заполнено', 'Толтырылмаған')
    default:
      return tr('Все статусы', 'Барлық мәртебелер')
  }
}

function copyReportFilters(filters) {
  return {
    statuses: Array.isArray(filters?.statuses)
      ? filters.statuses.map((value) => String(value).trim()).filter((value) => value && value !== 'not_filled')
      : [],
    responsible_ids: Array.isArray(filters?.responsible_ids) ? [...filters.responsible_ids] : [],
    directions: Array.isArray(filters?.directions) ? [...filters.directions] : [],
    period_from: String(filters?.period_from ?? ''),
    period_to: String(filters?.period_to ?? '')
  }
}

function selectAllReportResponsibles() {
  reportFilterDraft.value.responsible_ids = [...allReportResponsibleIds.value]
}

function clearReportResponsibles() {
  reportFilterDraft.value.responsible_ids = []
}

function selectAllReportStatuses() {
  reportFilterDraft.value.statuses = [...allReportStatusValues.value]
}

function clearReportStatuses() {
  reportFilterDraft.value.statuses = []
}

function openReportFilterModal() {
  if (!isAdmin.value) {
    return
  }
  clearMessages()
  reportExportMenuOpen.value = false
  reportFilterDraft.value = copyReportFilters(reportFilters.value)
  reportFilterModalOpen.value = true
}

function closeReportFilterModal() {
  reportFilterModalOpen.value = false
}

function resetReportFilters() {
  reportFilterDraft.value = createDefaultReportFilters()
}

function applyReportFilters() {
  const from = parseISODate(reportFilterDraft.value.period_from, false)
  const to = parseISODate(reportFilterDraft.value.period_to, true)
  if (from && to && from > to) {
    errorMessage.value = tr('Начало периода не может быть позже окончания', 'Период басы аяқталу күнінен кейін болмауы керек')
    return
  }
  reportFilters.value = copyReportFilters(reportFilterDraft.value)
  reportFilterModalOpen.value = false
}

function toggleReportExportMenu() {
  if (!isAdmin.value) {
    return
  }
  clearMessages()
  reportExportMenuOpen.value = !reportExportMenuOpen.value
}

function escapeHTML(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function htmlizeMultiline(value) {
  const normalized = String(value ?? '').trim()
  if (!normalized) {
    return '—'
  }
  return escapeHTML(normalized).replaceAll('\n', '<br>')
}

function buildExportIndicatorText(row) {
  const indicator = String(row?.development_indicator ?? '').trim() || '—'
  const plannedValue = formatPlannedValue(row?.planned_value, row?.measurement_unit || row?.unit)
  if (!plannedValue || plannedValue === '—') {
    return indicator
  }
  return `${indicator}\n(${plannedValue})`
}

function csvEscape(value) {
  const normalized = String(value ?? '').replaceAll('\r\n', '\n').replaceAll('\r', '\n')
  return `"${normalized.replaceAll('"', '""')}"`
}

function buildReportsExportCSV() {
  const header = [
    '№',
    tr('Индикатор', 'Индикатор'),
    tr('Мероприятия по достижению индикатора', 'Индикаторға қол жеткізу бойынша іс-шаралар'),
    tr('Срок исполнения', 'Орындау мерзімі'),
    tr('Ответственные', 'Жауаптылар')
  ]

  const lines = [
    header.map(csvEscape).join(';'),
    ...visibleReportRows.value.map((row, index) => [
      index + 1,
      buildExportIndicatorText(row),
      textPreview(row.report_preview),
      formatDateRange(row) || '—',
      textPreview(row.responsible)
    ].map(csvEscape).join(';'))
  ]

  return `\uFEFF${lines.join('\r\n')}`
}

function escapeRTF(value) {
  const source = String(value ?? '').replaceAll('\r\n', '\n').replaceAll('\r', '\n')
  let result = ''

  for (let index = 0; index < source.length; index += 1) {
    const char = source[index]
    const code = source.charCodeAt(index)

    if (char === '\\') {
      result += '\\\\'
      continue
    }
    if (char === '{') {
      result += '\\{'
      continue
    }
    if (char === '}') {
      result += '\\}'
      continue
    }
    if (char === '\n') {
      result += '\\line '
      continue
    }
    if (code > 127) {
      const signedCode = code > 0x7fff ? code - 0x10000 : code
      result += `\\u${signedCode}?`
      continue
    }

    result += char
  }

  return result
}

function buildRTFTableRow(cells, cellEdges, { header = false } = {}) {
  const edgeMarkup = cellEdges.map((edge) => `\\cellx${edge}`).join('')
  const cellContent = cells
    .map((cell) => `${header ? '\\b ' : ''}\\intbl ${escapeRTF(cell)}${header ? '\\b0 ' : ''}\\cell`)
    .join('')

  return `\\trowd\\trgaph108\\trleft0${edgeMarkup}${cellContent}\\row\n`
}

function buildReportsExportRTF() {
  const generatedAt = new Date().toLocaleString(tr('ru-RU', 'kk-KZ'))
  const filterLines = currentReportFilterSummary()
    .map((item) => `\\bullet\\tab ${escapeRTF(item)}\\par`)
    .join('\n')

  const cellEdges = [700, 6500, 11000, 13200, 17200]
  const headerRow = buildRTFTableRow([
    '№',
    tr('Индикатор', 'Индикатор'),
    tr('Мероприятия по достижению индикатора', 'Индикаторға қол жеткізу бойынша іс-шаралар'),
    tr('Срок исполнения', 'Орындау мерзімі'),
    tr('Ответственные', 'Жауаптылар')
  ], cellEdges, { header: true })

  const bodyRows = visibleReportRows.value
    .map((row, index) => buildRTFTableRow([
      String(index + 1),
      buildExportIndicatorText(row),
      textPreview(row.report_preview),
      formatDateRange(row) || '—',
      textPreview(row.responsible)
    ], cellEdges))
    .join('')

  return `{\\rtf1\\ansi\\deff0{\\fonttbl{\\f0 Arial;}}\\paperw16840\\paperh11907\\landscape\\margl720\\margr720\\margt720\\margb720
\\fs22
\\b ${escapeRTF(tr('Планы и отчеты: отчеты', 'Жоспарлар мен есептер: есептер'))}\\b0\\par
${escapeRTF(tr('Год', 'Жыл'))}: ${escapeRTF(selectedYear.value)} \\bullet ${escapeRTF(tr('Сформировано', 'Қалыптастырылған уақыт'))}: ${escapeRTF(generatedAt)}\\par
\\par
${filterLines}
\\par
${headerRow}${bodyRows}
}`
}

function currentReportFilterSummary() {
  const parts = []
  const filters = reportFilters.value

  if ((filters.statuses ?? []).length > 0) {
    parts.push(`${tr('Статус', 'Мәртебе')}: ${(filters.statuses ?? []).map((status) => reportFilterStatusLabel(status)).join(', ')}`)
  }

  if ((filters.responsible_ids ?? []).length > 0) {
    const responsibleNames = (filters.responsible_ids ?? [])
      .map((id) => prorectors.value.find((item) => String(item.id) === String(id)))
      .filter(Boolean)
      .map((item) => item.full_name)
    if (responsibleNames.length > 0) {
      parts.push(`${tr('Ответственные', 'Жауаптылар')}: ${responsibleNames.join(', ')}`)
    }
  }

  if ((filters.directions ?? []).length > 0) {
    const directionNames = (filters.directions ?? [])
      .map((value) => directionOptions.find((item) => item.value === value))
      .filter(Boolean)
      .map((item) => tr(item.ru, item.kz))
    if (directionNames.length > 0) {
      parts.push(`${tr('Направление', 'Бағыт')}: ${directionNames.join(', ')}`)
    }
  }

  if (filters.period_from || filters.period_to) {
    const from = filters.period_from ? formatISODateToDMY(filters.period_from) : '—'
    const to = filters.period_to ? formatISODateToDMY(filters.period_to) : '—'
    parts.push(`${tr('Период', 'Период')}: ${from} - ${to}`)
  }

  return parts.length > 0 ? parts : [tr('Без фильтров', 'Сүзгісіз')]
}

function buildReportsExportHTML() {
  const rowsToExport = visibleReportRows.value
  const generatedAt = new Date().toLocaleString(tr('ru-RU', 'kk-KZ'))
  const tableRows = rowsToExport.map((row, index) => `
    <tr>
      <td>${index + 1}</td>
      <td>${htmlizeMultiline(buildExportIndicatorText(row))}</td>
      <td>${htmlizeMultiline(row.report_preview)}</td>
      <td>${htmlizeMultiline(formatDateRange(row))}</td>
      <td>${htmlizeMultiline(row.responsible)}</td>
    </tr>
  `).join('')

  const summaryRows = currentReportFilterSummary()
    .map((item) => `<li>${escapeHTML(item)}</li>`)
    .join('')

  return `<!DOCTYPE html>
<html lang="${tr('ru', 'kk')}">
<head>
  <meta charset="UTF-8">
  <title>${escapeHTML(tr('Экспорт отчетов', 'Есептер экспорты'))}</title>
  <style>
    body { font-family: Arial, Helvetica, sans-serif; color: #10212a; margin: 24px; }
    h1 { margin: 0 0 8px; font-size: 24px; }
    .meta { margin: 0 0 18px; color: #52606d; font-size: 13px; }
    .filters { margin: 0 0 18px; padding-left: 18px; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th, td { border: 1px solid #cfd9e3; padding: 10px; vertical-align: top; text-align: left; font-size: 12px; line-height: 1.45; word-break: break-word; white-space: pre-wrap; }
    th { background: #f4eee4; color: #445e74; text-transform: uppercase; letter-spacing: 0.04em; }
    .col-num { width: 48px; }
    .col-deadline { width: 140px; }
    .col-responsible { width: 180px; }
    @media print {
      body { margin: 10mm; }
    }
  </style>
</head>
<body>
  <h1>${escapeHTML(tr('Планы и отчеты: отчеты', 'Жоспарлар мен есептер: есептер'))}</h1>
  <p class="meta">${escapeHTML(tr('Год', 'Жыл'))}: ${escapeHTML(selectedYear.value)} • ${escapeHTML(tr('Сформировано', 'Қалыптастырылған уақыт'))}: ${escapeHTML(generatedAt)}</p>
  <ul class="filters">${summaryRows}</ul>
  <table>
    <thead>
      <tr>
        <th class="col-num">№</th>
        <th>${escapeHTML(tr('Индикатор', 'Индикатор'))}</th>
        <th>${escapeHTML(tr('Мероприятия по достижению индикатора', 'Индикаторға қол жеткізу бойынша іс-шаралар'))}</th>
        <th class="col-deadline">${escapeHTML(tr('Срок исполнения', 'Орындау мерзімі'))}</th>
        <th class="col-responsible">${escapeHTML(tr('Ответственные', 'Жауаптылар'))}</th>
      </tr>
    </thead>
    <tbody>${tableRows}</tbody>
  </table>
</body>
</html>`
}

function downloadStringFile(content, mimeType, fileName) {
  const blob = new Blob([content], { type: mimeType })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}

function exportReports(format) {
  if (!isAdmin.value) {
    return
  }

  const rowsToExport = visibleReportRows.value
  if (rowsToExport.length === 0) {
    errorMessage.value = tr('Нет данных для выгрузки', 'Жүктеуге дерек жоқ')
    reportExportMenuOpen.value = false
    return
  }

  clearMessages()
  const html = buildReportsExportHTML()
  const timeSuffix = selectedYear.value || new Date().getFullYear()

  if (format === 'doc') {
    const rtf = buildReportsExportRTF()
    downloadStringFile(rtf, 'application/rtf;charset=utf-8', `plans-reports-${timeSuffix}.rtf`)
    successMessage.value = tr('Файл RTF выгружен', 'RTF файлы жүктелді')
    reportExportMenuOpen.value = false
    return
  }

  if (format === 'excel') {
    const csv = buildReportsExportCSV()
    downloadStringFile(csv, 'text/csv;charset=utf-8', `plans-reports-${timeSuffix}.csv`)
    successMessage.value = tr('Файл CSV для Excel выгружен', 'Excel үшін CSV файлы жүктелді')
    reportExportMenuOpen.value = false
    return
  }

  if (format === 'pdf') {
    const printWindow = window.open('', '_blank', 'width=1200,height=900')
    if (!printWindow) {
      errorMessage.value = tr('Не удалось открыть окно печати PDF', 'PDF басып шығару терезесін ашу мүмкін болмады')
      reportExportMenuOpen.value = false
      return
    }

    printWindow.document.open()
    printWindow.document.write(html)
    printWindow.document.close()
    printWindow.focus()
    printWindow.onload = () => {
      printWindow.print()
    }
    successMessage.value = tr('Открыто окно сохранения в PDF', 'PDF сақтауға арналған терезе ашылды')
  }

  reportExportMenuOpen.value = false
}

function formatDateTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '—'
  }
  return date.toLocaleString(tr('ru-RU', 'kk-KZ'))
}

async function downloadReportFile(file) {
  if (!file?.id) {
    return
  }

  clearMessages()
  try {
    const response = await downloadPlanReportFile(file.id)
    const blob = new Blob([response.data], {
      type: response.headers?.['content-type'] || 'application/octet-stream'
    })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.file_name || `report-file-${file.id}`
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось скачать файл', 'Файлды жүктеу мүмкін болмады')
  }
}

async function openReportsHistoryModal(row) {
  if (!row) {
    return
  }

  clearMessages()
  reportExportMenuOpen.value = false
  reportsHistoryIndicator.value = row
  reportsHistoryModalOpen.value = true
  await loadReportsHistory(row.indicator_id)
}

function closeReportsHistoryModal() {
  reportsHistoryModalOpen.value = false
  reportsHistoryIndicator.value = null
  reportsHistoryItems.value = []
  reportsHistoryLoading.value = false
  closeHistoryEditModal()
  closeHistoryReviewModal()
}

function openNewReportFromHistory() {
  if (!reportsHistoryIndicator.value) {
    return
  }
  const row = rows.value.find((item) => item.indicator_id === reportsHistoryIndicator.value.indicator_id)
  closeReportsHistoryModal()
  if (row) {
    openReportModal(row)
  }
}

async function loadReportsHistory(indicatorID = reportsHistoryIndicator.value?.indicator_id) {
  if (!indicatorID) {
    reportsHistoryItems.value = []
    return
  }

  reportsHistoryItems.value = []
  reportsHistoryLoading.value = true

  try {
    const response = await fetchPlanReports(selectedYear.value, {
      indicator_id: indicatorID
    })
    reportsHistoryItems.value = response.items ?? []
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось загрузить историю отчетов', 'Есеп тарихын жүктеу мүмкін болмады')
  } finally {
    reportsHistoryLoading.value = false
  }
}

function canReviewHistoryItem(item) {
  return String(item?.status ?? '').toLowerCase() === 'pending'
}

function openHistoryEditModal(item) {
  if (!isAdmin.value || !item) {
    return
  }

  clearMessages()
  historyEditReportId.value = item.id
  historyEditText.value = item.report_text ?? ''
  historyEditModalOpen.value = true
}

function closeHistoryEditModal() {
  historyEditModalOpen.value = false
  historyEditReportId.value = null
  historyEditText.value = ''
  historyEditSaving.value = false
}

async function submitHistoryEdit() {
  if (!isAdmin.value || historyEditReportId.value === null) {
    return
  }

  historyEditSaving.value = true
  clearMessages()

  try {
    await updatePlanReport(historyEditReportId.value, {
      report_text: historyEditText.value
    })
    successMessage.value = tr('Текст отчета обновлен', 'Есеп мәтіні жаңартылды')
    closeHistoryEditModal()
    await loadReportsHistory()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось обновить текст отчета', 'Есеп мәтінін жаңарту мүмкін болмады')
  } finally {
    historyEditSaving.value = false
  }
}

function openHistoryApproveModal(item) {
  if (!isAdmin.value || !item) {
    return
  }

  clearMessages()
  historyReviewMode.value = 'approve'
  historyReviewReportId.value = item.id
  historyReviewText.value = item.approval_formula ?? ''
  historyReviewModalOpen.value = true
}

function openHistoryRejectModal(item) {
  if (!isAdmin.value || !item) {
    return
  }

  clearMessages()
  historyReviewMode.value = 'reject'
  historyReviewReportId.value = item.id
  historyReviewText.value = item.review_note ?? ''
  historyReviewModalOpen.value = true
}

function closeHistoryReviewModal() {
  historyReviewModalOpen.value = false
  historyReviewReportId.value = null
  historyReviewText.value = ''
  historyReviewSaving.value = false
}

async function submitHistoryReview() {
  if (!isAdmin.value || historyReviewReportId.value === null) {
    return
  }

  const normalizedText = historyReviewText.value.trim()
  if (!normalizedText) {
    errorMessage.value = historyReviewMode.value === 'approve'
      ? tr('Заполните формулу', 'Формуланы толтырыңыз')
      : tr('Заполните причину отклонения', 'Қабылдамау себебін толтырыңыз')
    return
  }

  historyReviewSaving.value = true
  clearMessages()

  try {
    if (historyReviewMode.value === 'approve') {
      await reviewPlanReport(historyReviewReportId.value, {
        action: 'approve',
        approval_formula: normalizedText
      })
      successMessage.value = tr('Отчет принят', 'Есеп қабылданды')
    } else {
      await reviewPlanReport(historyReviewReportId.value, {
        action: 'reject',
        review_note: normalizedText
      })
      successMessage.value = tr('Отчет отклонен', 'Есеп қабылданбады')
    }

    closeHistoryReviewModal()
    await loadReportsHistory()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось обновить статус отчета', 'Есеп статусын жаңарту мүмкін болмады')
  } finally {
    historyReviewSaving.value = false
  }
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
    errorMessage.value = tr('Нужно загрузить минимум один файл', 'Кемінде бір файл жүктеу міндетті')
    return
  }

  reportSending.value = true
  clearMessages()

  try {
    await submitPlanIndicatorReport(row.indicator_id, selectedYear.value, {
      report_text: normalizedText,
      files: reportFiles.value
    })
    successMessage.value = tr(`Отчет по индикатору №${row.indicator_id} отправлен`, `Индикатор №${row.indicator_id} бойынша отчет жіберілді`)
    closeReportModal()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Ошибка отправки отчета', 'Отчет жіберу кезінде қате болды')
  } finally {
    reportSending.value = false
  }
}

onMounted(() => {
  startCountdownTicker()
  if (!canUseReportsTab.value) {
    activeViewTab.value = 'plans'
  }
  initialize()
})

onBeforeUnmount(() => {
  stopCountdownTicker()
})
</script>

<template>
  <section class="plans-page">
    <PageHeader
      :title="tr('Планы и отчеты', 'Жоспарлар мен есептер')"
      :subtitle="tr('Рабочая матрица текущего года: мероприятия, сроки исполнения и ответственные по каждому индикатору', 'Ағымдағы жыл матрицасы: әр индикатор үшін іс-шаралар, мерзімдер және жауаптылар')"
      :eyebrow="tr('Сетка исполнения', 'Орындау кестесі')"
    />

    <div class="panel panel-strong toolbar-panel plans-toolbar">
      <div class="plans-year-card">
        <span class="kicker">{{ tr('Текущий год', 'Ағымдағы жыл') }}</span>
        <strong>{{ selectedYear }}</strong>
        <p>{{ tr('Раздел синхронизирован только с индикаторами текущего года.', 'Бұл бөлім тек ағымдағы жыл индикаторларымен синхрондалады.') }}</p>
      </div>

      <label v-if="activeViewTab === 'plans' && isAdmin" class="plans-filter">
        <span>{{ tr('Ответственные', 'Жауаптылар') }}</span>
        <select v-model="selectedResponsibleFilter">
          <option value="">{{ tr('Все', 'Барлығы') }}</option>
          <option v-for="prorector in prorectors" :key="`filter-${prorector.id}`" :value="String(prorector.id)">
            {{ prorector.full_name }}
          </option>
        </select>
      </label>

      <label v-if="activeViewTab === 'plans'" class="plans-filter">
        <span>{{ tr('Направление', 'Бағыт') }}</span>
        <select v-model="selectedDirectionFilter">
          <option value="">{{ tr('Все направления', 'Барлық бағыттар') }}</option>
          <option v-for="option in directionOptions" :key="`direction-filter-${option.value}`" :value="option.value">
            {{ tr(option.ru, option.kz) }}
          </option>
        </select>
      </label>

      <div class="plans-visible-card">
        <span class="kicker">{{ tr('Отображено', 'Көрсетілген') }}</span>
        <strong>{{ activeVisibleCount }}</strong>
      </div>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <section class="panel panel-strong plans-table-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ tr('Операционный план', 'Операциялық жоспар') }}</h3>
          <p class="panel-subtitle">{{ activePanelSubtitle }}</p>
        </div>
        <div class="plans-header-actions">
          <div v-if="isAdmin && activeViewTab === 'reports'" class="plans-reports-actions">
            <button class="btn btn-ghost plans-toolbar-btn" type="button" @click="openReportFilterModal">
              {{ tr('Фильтр', 'Сүзгі') }}
            </button>
            <div class="plans-export-wrap">
              <button class="btn btn-primary plans-toolbar-btn" type="button" @click="toggleReportExportMenu">
                {{ tr('Выгрузить', 'Жүктеу') }}
              </button>
              <div v-if="reportExportMenuOpen" class="plans-export-menu">
                <button class="btn btn-ghost plans-export-option" type="button" @click="exportReports('doc')">RTF</button>
                <button class="btn btn-ghost plans-export-option" type="button" @click="exportReports('excel')">EXCEL</button>
                <button class="btn btn-ghost plans-export-option" type="button" @click="exportReports('pdf')">PDF</button>
              </div>
            </div>
          </div>
          <span class="kicker">{{ activeVisibleCount }} {{ tr('индикаторов', 'индикатор') }}</span>
        </div>
      </div>

      <div class="plans-card-tabs" :class="{ 'is-single': !canUseReportsTab }">
        <span
          v-if="canUseReportsTab"
          class="plans-card-tab-slider"
          :class="{ 'is-reports': activeViewTab === 'reports' }"
        />
        <button
          class="btn btn-ghost plans-card-tab-btn"
          :class="{ 'is-active': activeViewTab === 'plans' }"
          type="button"
          @click="setActiveViewTab('plans')"
        >
          {{ tr('Планы', 'Жоспарлар') }}
        </button>
        <button
          v-if="canUseReportsTab"
          class="btn btn-ghost plans-card-tab-btn"
          :class="{ 'is-active': activeViewTab === 'reports' }"
          type="button"
          @click="setActiveViewTab('reports')"
        >
          {{ tr('Отчеты', 'Есептер') }}
        </button>
      </div>

      <Transition :name="tabTransitionName" mode="out-in">
        <div v-if="activeViewTab === 'plans'" key="plans" class="plans-tab-pane">
          <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
          <div v-else-if="!hasRows" class="empty-state">
            {{ tr(`По году ${selectedYear} нет запланированных индикаторов.`, `${selectedYear} жылына жоспарланған индикатор табылмады.`) }}
          </div>
          <div v-else-if="!hasVisibleRows" class="empty-state">
            {{ tr('По выбранным фильтрам индикаторы не найдены.', 'Таңдалған сүзгілер бойынша индикатор табылмады.') }}
          </div>

          <div v-else class="table-wrapper">
            <table class="plan-table">
            <thead>
              <tr>
                <th class="col-number">№</th>
                <th>{{ tr('Индикатор Программы развития', 'Бағдарлама дамуының индикаторы') }}</th>
                <th class="col-deadline">{{ tr('Срок исполнения', 'Орындау мерзімі') }}</th>
                <th class="col-responsible">{{ tr('Ответственные', 'Жауаптылар') }}</th>
              </tr>
              </thead>
              <tbody>
                <tr v-for="(row, index) in visibleRows" :key="row.indicator_id">
                  <td class="number-cell" data-label="№">{{ index + 1 }}</td>

                  <td :data-label="tr('Индикатор программы развития', 'Бағдарлама индикаторы')">
                    <div
                      class="table-text-preview text-pretty"
                      :class="{ 'is-empty': textPreview(row.development_indicator) === '—' }"
                      role="button"
                      tabindex="0"
                      @click="openReadModal(tr('Индикатор Программы развития', 'Бағдарлама индикаторы'), row.development_indicator)"
                      @keyup.enter="openReadModal(tr('Индикатор Программы развития', 'Бағдарлама индикаторы'), row.development_indicator)"
                      @keyup.space.prevent="openReadModal(tr('Индикатор Программы развития', 'Бағдарлама индикаторы'), row.development_indicator)"
                    >
                      <span class="table-text-preview-content">{{ textPreview(row.development_indicator) }}</span>
                    </div>
                  <span class="planned-value-chip">
                    {{ formatPlannedValue(row.planned_value, row.measurement_unit || row.unit) }}
                  </span>
                </td>

                <td :data-label="tr('Срок исполнения', 'Орындау мерзімі')">
                  <div class="plans-schedule-card">
                      <p class="table-inline-value">{{ formatDateRange(row) || '—' }}</p>
                      <div class="schedule-status" :class="`schedule-${row.schedule_status}`">
                        {{ scheduleStatusLabel(row.schedule_status) }}
                      </div>
                      <div v-if="row.execution_start_date && row.execution_end_date" class="countdown-text">
                        {{ tr('Осталось времени:', 'Қалған уақыт:') }} {{ formatRemainingTime(row) }}
                      </div>
                    </div>
                  </td>

                  <td :data-label="tr('Ответственные', 'Жауаптылар')">
                    <template v-if="isAdmin">
                      <p class="table-inline-value text-pretty">
                        {{ row.responsible || tr('Ответственные не выбраны', 'Жауаптылар таңдалмаған') }}
                      </p>
                      <button
                        class="btn btn-primary plans-edit-row-btn"
                        type="button"
                        @click="openRowEditModal(row)"
                      >
                        {{ tr('Изменить', 'Өзгерту') }}
                      </button>
                    </template>
                    <template v-else>
                      <p class="table-inline-value text-pretty">{{ row.responsible || '—' }}</p>
                      <button
                        v-if="isProrector"
                        class="btn btn-primary plans-report-btn"
                        type="button"
                        @click="openReportModal(row)"
                      >
                        {{ tr('Отправить отчет', 'Есеп жіберу') }}
                      </button>
                    </template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-else key="reports" class="plans-tab-pane">
          <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
          <div v-else-if="!hasRows" class="empty-state">
            {{ tr(`По году ${selectedYear} нет индикаторов.`, `${selectedYear} жылына индикаторлар табылмады.`) }}
          </div>
          <div v-else-if="!hasReportBaseRows" class="empty-state">
            {{
              tr(
                'Для раздела отчётов пока нет полностью заполненных индикаторов.',
                'Есептер бөлімі үшін әзірге толық толтырылған индикаторлар жоқ.'
              )
            }}
          </div>
          <div v-else-if="!hasVisibleReportRows" class="empty-state">
            {{ tr('По выбранным фильтрам индикаторы не найдены.', 'Таңдалған сүзгілер бойынша индикатор табылмады.') }}
          </div>
          <div v-else class="table-wrapper">
            <table class="plan-table plans-reports-table">
              <thead>
                <tr>
                  <th class="col-number">№</th>
                  <th>{{ tr('Индикатор', 'Индикатор') }}</th>
                  <th class="col-reports-preview">{{ tr('Мероприятия по достижению индикатора', 'Индикаторға қол жеткізу бойынша іс-шаралар') }}</th>
                  <th class="col-deadline-compact">{{ tr('Срок исполнения', 'Орындау мерзімі') }}</th>
                  <th class="col-responsible">{{ tr('Ответственные', 'Жауаптылар') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, index) in visibleReportRows" :key="`report-list-${row.indicator_id}`">
                  <td class="number-cell" data-label="№">{{ index + 1 }}</td>
                  <td :data-label="tr('Индикатор', 'Индикатор')">
                    <div
                      class="table-text-preview text-pretty"
                      :class="{ 'is-empty': textPreview(row.development_indicator) === '—' }"
                      role="button"
                      tabindex="0"
                      @click="openReadModal(tr('Индикатор', 'Индикатор'), row.development_indicator)"
                      @keyup.enter="openReadModal(tr('Индикатор', 'Индикатор'), row.development_indicator)"
                      @keyup.space.prevent="openReadModal(tr('Индикатор', 'Индикатор'), row.development_indicator)"
                    >
                      <span class="table-text-preview-content">{{ textPreview(row.development_indicator) }}</span>
                    </div>
                    <div class="plans-report-row-meta">
                      <span class="planned-value-chip">
                        {{ formatPlannedValue(row.planned_value, row.measurement_unit || row.unit) }}
                      </span>
                      <button
                        class="btn btn-ghost plans-inline-icon-btn"
                        type="button"
                        :aria-label="tr('Открыть отчеты', 'Есептерді ашу')"
                        @click="openReportsHistoryModal(row)"
                      >
                        <svg viewBox="0 0 24 24" aria-hidden="true">
                          <path d="M9 6h8m-8 6h8m-8 6h5M7 4h10a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z" />
                        </svg>
                      </button>
                    </div>
                  </td>
                  <td :data-label="tr('Мероприятия по достижению индикатора', 'Индикаторға қол жеткізу бойынша іс-шаралар')">
                    <div
                      class="table-text-preview text-pretty reports-preview-cell"
                      :class="{ 'is-empty': textPreview(row.report_preview) === '—' }"
                      role="button"
                      tabindex="0"
                      @click="openReadModal(tr('Тексты отчетов', 'Есеп мәтіндері'), row.report_preview)"
                      @keyup.enter="openReadModal(tr('Тексты отчетов', 'Есеп мәтіндері'), row.report_preview)"
                      @keyup.space.prevent="openReadModal(tr('Тексты отчетов', 'Есеп мәтіндері'), row.report_preview)"
                    >
                      <span class="table-text-preview-content">{{ textPreview(row.report_preview) }}</span>
                    </div>
                  </td>
                  <td :data-label="tr('Срок исполнения', 'Орындау мерзімі')">
                    <div class="plans-schedule-card plans-schedule-card-compact">
                      <p class="table-inline-value">{{ formatDateRange(row) || '—' }}</p>
                    </div>
                  </td>
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
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </Transition>
    </section>

    <div v-if="reportFilterModalOpen" class="modal-backdrop" @click.self="closeReportFilterModal">
      <div class="modal-card plans-modal plans-filter-modal">
        <h3 class="modal-title">{{ tr('Фильтры для отчётов', 'Есептерге арналған сүзгілер') }}</h3>
        <p class="modal-subtitle">
          {{ tr('Статус, ответственные, направление и период влияют на таблицу и выгрузку.', 'Мәртебе, жауаптылар, бағыт және период кесте мен экспортқа бірдей әсер етеді.') }}
        </p>

        <div class="row-edit-grid">
          <label class="modal-label">
            <span>{{ tr('Статус', 'Мәртебе') }}</span>
            <div class="report-filter-head">
              <div class="report-filter-actions">
                <button class="btn btn-ghost report-filter-action-btn" type="button" @click="selectAllReportStatuses">
                  {{ tr('Все', 'Барлығы') }}
                </button>
                <button class="btn btn-ghost report-filter-action-btn" type="button" @click="clearReportStatuses">
                  {{ tr('Снять', 'Тазарту') }}
                </button>
              </div>
              <span class="report-filter-meta">
                {{ tr('Выбрано:', 'Таңдалды:') }} {{ reportStatusSelectionCount }} / {{ reportStatusOptions.length }}
              </span>
            </div>
            <div class="report-filter-group">
              <label
                v-for="option in reportStatusOptions"
                :key="`report-status-${option.value}`"
                class="report-filter-option"
                :class="{ 'is-selected': reportFilterDraft.statuses.includes(option.value) }"
              >
                <input
                  v-model="reportFilterDraft.statuses"
                  type="checkbox"
                  :value="option.value"
                />
                <span>{{ tr(option.ru, option.kz) }}</span>
              </label>
            </div>
          </label>

          <label class="modal-label">
            <span>{{ tr('Ответственные', 'Жауаптылар') }}</span>
            <div class="report-filter-head">
              <div class="report-filter-actions">
                <button class="btn btn-ghost report-filter-action-btn" type="button" @click="selectAllReportResponsibles">
                  {{ tr('Все', 'Барлығы') }}
                </button>
                <button class="btn btn-ghost report-filter-action-btn" type="button" @click="clearReportResponsibles">
                  {{ tr('Снять', 'Тазарту') }}
                </button>
              </div>
              <span class="report-filter-meta">
                {{ tr('Выбрано:', 'Таңдалды:') }} {{ reportResponsibleSelectionCount }} / {{ prorectors.length }}
              </span>
            </div>
            <div class="report-filter-group report-filter-group-scroll">
              <label
                v-for="prorector in prorectors"
                :key="`report-filter-${prorector.id}`"
                class="report-filter-option"
                :class="{ 'is-selected': reportFilterDraft.responsible_ids.includes(String(prorector.id)) }"
              >
                <input
                  v-model="reportFilterDraft.responsible_ids"
                  type="checkbox"
                  :value="String(prorector.id)"
                />
                <span>{{ prorector.full_name }}</span>
              </label>
            </div>
          </label>

          <label class="modal-label">
            <span>{{ tr('Направление', 'Бағыт') }}</span>
            <div class="report-filter-group">
              <label
                v-for="option in directionOptions"
                :key="`report-direction-${option.value}`"
                class="report-filter-option"
                :class="{ 'is-selected': reportFilterDraft.directions.includes(option.value) }"
              >
                <input
                  v-model="reportFilterDraft.directions"
                  type="checkbox"
                  :value="option.value"
                />
                <span>{{ tr(option.ru, option.kz) }}</span>
              </label>
            </div>
          </label>

          <div class="row-edit-dates">
            <div class="date-range-grid">
              <label class="date-field">
                <span>{{ tr('Период с', 'Период басы') }}</span>
                <input v-model="reportFilterDraft.period_from" class="plans-input" type="date" />
              </label>
              <label class="date-field">
                <span>{{ tr('Период по', 'Период соңы') }}</span>
                <input v-model="reportFilterDraft.period_to" class="plans-input" type="date" />
              </label>
            </div>
          </div>
        </div>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="resetReportFilters">
            {{ tr('Сбросить', 'Тазарту') }}
          </button>
          <button class="btn btn-ghost" type="button" @click="closeReportFilterModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button class="btn btn-primary" type="button" @click="applyReportFilters">
            {{ tr('Применить', 'Қолдану') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="reportsHistoryModalOpen" class="modal-backdrop" @click.self="closeReportsHistoryModal">
      <div class="modal-card plans-modal plans-history-modal">
        <h3 class="modal-title">{{ tr('История отчетов', 'Есептер тарихы') }}</h3>
        <p class="modal-subtitle text-pretty">
          {{ reportsHistoryIndicator?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <div class="plans-history-toolbar">
          <span class="kicker">{{ tr('Отчетов', 'Есептер') }}: {{ reportsHistoryItems.length }}</span>
          <button
            v-if="isProrector"
            class="btn btn-primary"
            type="button"
            @click="openNewReportFromHistory"
          >
            {{ tr('Отправить новый отчет', 'Жаңа есеп жіберу') }}
          </button>
        </div>

        <div v-if="reportsHistoryLoading" class="empty-state plans-history-empty">
          {{ tr('Загрузка...', 'Жүктелуде...') }}
        </div>
        <div v-else-if="reportsHistoryItems.length === 0" class="empty-state plans-history-empty">
          {{ tr('По выбранному индикатору отчетов пока нет.', 'Таңдалған индикатор бойынша әзірге есептер жоқ.') }}
        </div>
        <div v-else class="plans-history-list">
          <article
            v-for="item in reportsHistoryItems"
            :key="`history-item-${item.id}`"
            class="plans-history-item"
          >
            <div class="plans-history-item-head">
              <div class="plans-history-item-meta">
                <span class="planned-value-chip">#{{ item.id }}</span>
                <span class="schedule-status" :class="reportStatusClass(item.status)">
                  {{ reportStatusLabel(item.status) }}
                </span>
              </div>
              <button
                v-if="isAdmin"
                class="btn btn-ghost plans-history-icon-btn"
                type="button"
                :aria-label="tr('Редактировать текст отчета', 'Есеп мәтінін өзгерту')"
                @click="openHistoryEditModal(item)"
              >
                ✎
              </button>
            </div>

            <div
              class="table-text-preview text-pretty plans-history-text"
              :class="{ 'is-empty': textPreview(item.report_text) === '—' }"
              role="button"
              tabindex="0"
              @click="openReadModal(tr('Текст отчета', 'Есеп мәтіні'), item.report_text)"
              @keyup.enter="openReadModal(tr('Текст отчета', 'Есеп мәтіні'), item.report_text)"
              @keyup.space.prevent="openReadModal(tr('Текст отчета', 'Есеп мәтіні'), item.report_text)"
            >
              <span class="table-text-preview-content">{{ textPreview(item.report_text) }}</span>
            </div>

            <div class="files-list plans-history-files">
              <button
                v-for="file in item.files"
                :key="`history-file-${file.id}`"
                class="btn btn-ghost report-file-chip"
                type="button"
                @click="downloadReportFile(file)"
              >
                {{ file.file_name }}
              </button>
              <span v-if="!item.files || item.files.length === 0" class="muted">
                {{ tr('Файлы не прикреплены', 'Файлдар тіркелмеген') }}
              </span>
            </div>

            <p class="meta">
              {{ tr('Отправил:', 'Жіберген:') }} {{ item.submitted_by_name || item.submitted_by }} •
              {{ formatDateTime(item.submitted_at) }}
            </p>
            <p v-if="item.reviewed_by_name" class="meta">
              {{ tr('Проверил:', 'Тексерген:') }} {{ item.reviewed_by_name }} •
              {{ formatDateTime(item.reviewed_at) }}
            </p>
            <p v-if="item.review_note" class="cell-note danger-text">
              {{ tr('Причина отклонения:', 'Қабылданбау себебі:') }} {{ item.review_note }}
            </p>
            <p v-if="item.approval_formula" class="cell-note success-text">
              {{ tr('Формула/итог:', 'Формула/қорытынды:') }} {{ item.approval_formula }}
            </p>

            <div v-if="isAdmin && canReviewHistoryItem(item)" class="plans-history-actions">
              <button class="btn btn-primary" type="button" @click="openHistoryApproveModal(item)">
                {{ tr('Принять', 'Қабылдау') }}
              </button>
              <button class="btn btn-danger" type="button" @click="openHistoryRejectModal(item)">
                {{ tr('Отклонить', 'Қабылдамау') }}
              </button>
            </div>
          </article>
        </div>

        <div class="modal-actions">
          <button class="btn btn-primary" type="button" @click="closeReportsHistoryModal">
            {{ tr('Закрыть', 'Жабу') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="historyEditModalOpen" class="modal-backdrop" @click.self="closeHistoryEditModal">
      <div class="modal-card plans-modal plans-edit-report-modal">
        <h3 class="modal-title">{{ tr('Редактировать текст отчета', 'Есеп мәтінін өзгерту') }}</h3>
        <p class="modal-subtitle text-pretty">
          {{ activeHistoryReport?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <label class="modal-label">
          <span>{{ tr('Текст отчета', 'Есеп мәтіні') }}</span>
          <textarea
            v-model="historyEditText"
            class="modal-textarea"
            rows="8"
            :placeholder="tr('Введите обновленный текст отчета...', 'Жаңартылған есеп мәтінін енгізіңіз...')"
          />
        </label>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeHistoryEditModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button class="btn btn-primary" type="button" :disabled="historyEditSaving" @click="submitHistoryEdit">
            {{ historyEditSaving ? tr('Сохранение...', 'Сақталуда...') : tr('Сохранить', 'Сақтау') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="historyReviewModalOpen" class="modal-backdrop" @click.self="closeHistoryReviewModal">
      <div class="modal-card plans-modal plans-review-modal">
        <h3 class="modal-title">
          {{ historyReviewMode === 'approve' ? tr('Принять отчет', 'Есепті қабылдау') : tr('Отклонить отчет', 'Есепті қабылдамау') }}
        </h3>
        <p class="modal-subtitle text-pretty">
          {{ activeHistoryReviewReport?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <label class="modal-label">
          <span v-if="historyReviewMode === 'approve'">
            {{ tr('Формула и итоговое значение', 'Формула және қорытынды сан') }}
          </span>
          <span v-else>
            {{ tr('Причина отклонения', 'Қабылдамау себебі') }}
          </span>
          <textarea
            v-model="historyReviewText"
            class="modal-textarea"
            rows="6"
            :placeholder="historyReviewMode === 'approve'
              ? tr('Например: 157/525 ППС + 11587 обучающихся *100%=1,3%', 'Мысалы: 157/525 ППС + 11587 обучающихся *100%=1,3%')
              : tr('Укажите причину отклонения...', 'Қабылданбау себебін жазыңыз...')"
          />
        </label>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeHistoryReviewModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button class="btn btn-primary" type="button" :disabled="historyReviewSaving" @click="submitHistoryReview">
            {{
              historyReviewSaving
                ? tr('Сохранение...', 'Сақталуда...')
                : historyReviewMode === 'approve'
                  ? tr('Принять', 'Қабылдау')
                  : tr('Отклонить', 'Қабылдамау')
            }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="readModalOpen" class="modal-backdrop" @click.self="closeReadModal">
      <div class="modal-card plans-modal plans-read-modal">
        <h3 class="modal-title">{{ readModalTitle }}</h3>
        <div class="plans-read-content text-pretty">
          {{ readModalText }}
        </div>

        <div class="modal-actions">
          <button class="btn btn-primary" type="button" @click="closeReadModal">
            {{ tr('Закрыть', 'Жабу') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="rowEditModalOpen" class="modal-backdrop" @click.self="closeRowEditModal">
      <div class="modal-card plans-modal plans-row-edit-modal">
        <h3 class="modal-title">{{ tr('Редактировать индикатор', 'Индикаторды өзгерту') }}</h3>
        <p class="modal-subtitle">
          {{ activeRowEdit?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <div class="row-edit-grid">
          <label class="modal-label">
            {{ tr('Индикатор Программы развития', 'Бағдарлама индикаторы') }}
            <div
              class="plans-readonly-indicator text-pretty"
              role="button"
              tabindex="0"
              @click="openReadModal(tr('Индикатор Программы развития', 'Бағдарлама индикаторы'), activeRowEdit?.development_indicator)"
              @keyup.enter="openReadModal(tr('Индикатор Программы развития', 'Бағдарлама индикаторы'), activeRowEdit?.development_indicator)"
              @keyup.space.prevent="openReadModal(tr('Индикатор Программы развития', 'Бағдарлама индикаторы'), activeRowEdit?.development_indicator)"
            >
              {{ textPreview(activeRowEdit?.development_indicator) }}
            </div>
          </label>

          <div class="row-edit-dates">
            <div class="date-range-grid">
              <label class="date-field">
                <span>{{ tr('Начало', 'Басталуы') }}</span>
                <input v-model="rowEditForm.execution_start_date" class="plans-input" type="date" />
              </label>
              <label class="date-field">
                <span>{{ tr('Окончание', 'Аяқталуы') }}</span>
                <input v-model="rowEditForm.execution_end_date" class="plans-input" type="date" />
              </label>
            </div>
            <p class="date-range-preview">
              {{ formatDateRange(rowEditForm) || '—' }}
            </p>
          </div>

          <label class="modal-label">
            <div class="row-edit-prorectors-head">
              <span>{{ tr('Ответственные', 'Жауаптылар') }}</span>
              <span class="row-edit-prorectors-meta">{{ tr('Выбрано:', 'Таңдалды:') }} {{ rowEditForm.responsible_user_ids.length }}</span>
            </div>
            <div class="row-edit-prorectors">
              <div class="prorector-list">
                <label
                  v-for="prorector in prorectors"
                  :key="`row-edit-prorector-${prorector.id}`"
                  class="prorector-item"
                  :class="{ 'is-selected': rowEditForm.responsible_user_ids.includes(Number(prorector.id)) }"
                >
                  <input
                    v-model="rowEditForm.responsible_user_ids"
                    type="checkbox"
                    :value="Number(prorector.id)"
                  />
                  <span>
                    <strong>{{ prorector.full_name }}</strong>
                    <small>@{{ prorector.username }}</small>
                  </span>
                </label>
              </div>
              <p v-if="prorectors.length === 0" class="empty-state row-edit-prorectors-empty">
                {{ tr('Список проректоров пуст.', 'Проректорлар тізімі жоқ.') }}
              </p>
            </div>
          </label>
        </div>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeRowEditModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="savingIndicatorId === rowEditIndicatorId"
            @click="saveRowFromModal"
          >
            {{ savingIndicatorId === rowEditIndicatorId ? tr('Сохранение...', 'Сақталуда...') : tr('Сохранить', 'Сақтау') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="reportModalOpen" class="modal-backdrop" @click.self="closeReportModal">
      <div class="modal-card plans-modal">
        <h3 class="modal-title">{{ tr('Отправить отчет', 'Есеп жіберу') }}</h3>
        <p class="modal-subtitle">
          {{ activeReportRow?.development_indicator || tr('Индикатор', 'Индикатор') }}
        </p>
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <label class="modal-label">
          {{ tr('Текст отчета', 'Есеп мәтіні') }}
          <textarea
            v-model="reportText"
            class="report-textarea"
            rows="6"
            :placeholder="tr('Опишите результат выполнения по индикатору...', 'Индикатор бойынша орындалу нәтижесін жазыңыз...')"
          />
        </label>

        <label class="modal-label">
          {{ tr('Документы (минимум 1 файл)', 'Құжаттар (кемінде 1 файл)') }}
          <input
            class="report-file-input"
            type="file"
            accept=".doc,.docx,.xls,.xlsx,.ppt,.pptx,.pdf"
            multiple
            @change="handleReportFileChange"
          />
        </label>
        <p v-if="reportFiles.length > 0" class="file-list">
          {{ tr('Выбранные файлы:', 'Таңдалған файлдар:') }} {{ reportFiles.map((file) => file.name).join(', ') }}
        </p>

        <div class="modal-actions">
          <button class="btn btn-ghost" type="button" @click="closeReportModal">
            {{ tr('Отмена', 'Бас тарту') }}
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="reportSending"
            @click="submitIndicatorReport"
          >
            {{ reportSending ? tr('Отправка...', 'Жіберілуде...') : tr('Отправить', 'Жіберу') }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.plans-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.9rem;
  align-items: end;
  justify-content: space-between;
}

.plans-year-card,
.plans-visible-card {
  display: grid;
  gap: 0.35rem;
}

.plans-year-card {
  flex: 1 1 18rem;
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
  min-width: 16rem;
  max-width: 24rem;
  flex: 1 1 16rem;
}

.plans-visible-card {
  margin-left: auto;
}

.plans-table-card {
  padding: 1.2rem;
}

.plans-card-tabs {
  position: relative;
  isolation: isolate;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.35rem;
  margin: 0.9rem 0 1rem;
  padding: 0.25rem;
  max-width: 22rem;
  border: 1px solid rgba(16, 33, 42, 0.12);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.7);
  overflow: hidden;
}

.plans-card-tabs.is-single {
  grid-template-columns: 1fr;
  max-width: 12rem;
}

.plans-card-tab-slider {
  position: absolute;
  top: 0.25rem;
  bottom: 0.25rem;
  left: 0.25rem;
  width: calc(50% - 0.175rem);
  border-radius: 11px;
  background: linear-gradient(125deg, rgba(17, 120, 111, 0.19), rgba(17, 120, 111, 0.09));
  box-shadow: 0 8px 20px rgba(15, 31, 41, 0.12);
  transition: transform 0.32s cubic-bezier(0.2, 0.75, 0.2, 1);
  z-index: 0;
}

.plans-card-tab-slider.is-reports {
  transform: translateX(calc(100% + 0.35rem));
}

.plans-card-tab-btn {
  min-width: 0;
  width: 100%;
  justify-content: center;
  position: relative;
  z-index: 1;
  border-color: transparent;
  background: transparent;
  transition: color 0.2s ease;
}

.plans-card-tab-btn.is-active {
  color: #0f5e57;
  border-color: transparent;
  background: transparent;
}

.plans-card-tab-btn:not(.is-active) {
  color: var(--muted-strong);
}

.plans-tab-pane {
  min-height: 6.4rem;
}

.plans-header-actions {
  display: flex;
  align-items: center;
  gap: 0.8rem;
  margin-left: auto;
}

.plans-reports-actions {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.plans-toolbar-btn {
  min-width: 8.5rem;
  justify-content: center;
}

.plans-export-wrap {
  position: relative;
}

.plans-export-menu {
  position: absolute;
  top: calc(100% + 0.5rem);
  left: 0;
  z-index: 12;
  min-width: 8.5rem;
  display: grid;
  gap: 0.4rem;
  padding: 0.55rem;
  border: 1px solid rgba(16, 33, 42, 0.12);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 18px 34px rgba(16, 33, 42, 0.12);
  backdrop-filter: blur(12px);
}

.plans-export-option {
  justify-content: center;
}

.plans-slide-left-enter-active,
.plans-slide-left-leave-active,
.plans-slide-right-enter-active,
.plans-slide-right-leave-active {
  transition: transform 0.28s ease, opacity 0.28s ease;
}

.plans-slide-left-enter-from {
  opacity: 0;
  transform: translateX(26px);
}

.plans-slide-left-leave-to {
  opacity: 0;
  transform: translateX(-26px);
}

.plans-slide-right-enter-from {
  opacity: 0;
  transform: translateX(-26px);
}

.plans-slide-right-leave-to {
  opacity: 0;
  transform: translateX(26px);
}

.plan-table {
  min-width: 860px;
}

.plans-reports-table {
  min-width: 1180px;
}

.col-number {
  width: 4rem;
}

.col-deadline {
  width: 16rem;
}

.col-responsible {
  width: 15rem;
}

.col-reports-preview {
  width: 23rem;
}

.col-deadline-compact {
  width: 12.5rem;
}

.number-cell {
  text-align: center;
  font-weight: 700;
}

.plans-textarea,
.plans-input {
  width: 100%;
}

.plans-readonly-indicator {
  min-height: 6.2rem;
  padding: 0.78rem 0.9rem;
  border: 1px solid rgba(16, 33, 42, 0.14);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.82);
  color: var(--text);
  line-height: 1.5;
  cursor: pointer;
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

.plans-edit-row-btn,
.plans-report-btn {
  width: 100%;
  justify-content: center;
  margin-top: 0.65rem;
  padding-inline: 0.85rem;
  text-align: center;
  line-height: 1.2;
  white-space: normal;
}

.plans-history-btn {
  width: 100%;
  justify-content: center;
  padding-inline: 0.85rem;
  text-align: center;
  line-height: 1.2;
  white-space: normal;
}

.plans-report-row-meta {
  margin-top: 0.65rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
}

.plans-inline-icon-btn {
  width: 2.45rem;
  min-width: 2.45rem;
  height: 2.45rem;
  border-radius: 999px;
  padding: 0;
  justify-content: center;
  border: 1px solid rgba(17, 120, 111, 0.16);
  background: rgba(233, 245, 243, 0.72);
  color: #0f5e57;
}

.plans-inline-icon-btn svg {
  width: 1.1rem;
  height: 1.1rem;
  stroke: currentColor;
  fill: none;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.reports-preview-cell .table-text-preview-content {
  -webkit-line-clamp: 5;
}

.plans-schedule-card-compact {
  gap: 0.3rem;
}

.cell-note {
  margin: 0.6rem 0 0;
  color: var(--muted);
  font-size: 0.82rem;
}

.danger-text {
  color: #a63f32;
}

.success-text {
  color: #0f5e57;
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

.plans-history-modal {
  width: min(900px, 100%);
}

.plans-filter-modal {
  width: min(760px, 100%);
}

.report-filter-group {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.55rem;
  margin-top: 0.15rem;
}

.report-filter-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  margin-top: 0.15rem;
}

.report-filter-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.report-filter-action-btn {
  min-width: 5.5rem;
  justify-content: center;
}

.report-filter-meta {
  color: var(--muted);
  font-size: 0.82rem;
  font-weight: 600;
}

.report-filter-group-scroll {
  max-height: min(38vh, 320px);
  overflow: auto;
  padding-right: 0.2rem;
}

.report-filter-option {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  gap: 0.6rem;
  padding: 0.72rem 0.82rem;
  border: 1px solid rgba(16, 33, 42, 0.12);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.92);
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease, box-shadow 0.2s ease;
}

.report-filter-option:hover {
  border-color: rgba(17, 120, 111, 0.38);
  box-shadow: 0 8px 24px rgba(15, 31, 41, 0.08);
}

.report-filter-option.is-selected {
  border-color: rgba(17, 120, 111, 0.52);
  background: linear-gradient(145deg, rgba(17, 120, 111, 0.12), rgba(17, 120, 111, 0.05));
}

.report-filter-option input {
  width: 1.02rem;
  height: 1.02rem;
  margin-top: 0.08rem;
  accent-color: var(--accent);
}

.report-filter-option span {
  line-height: 1.4;
}

@media (max-width: 720px) {
  .report-filter-head {
    align-items: flex-start;
  }

  .report-filter-meta {
    width: 100%;
  }
}

.plans-edit-report-modal,
.plans-review-modal {
  width: min(720px, 100%);
}

.plans-history-toolbar {
  margin-top: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
}

.plans-history-empty {
  margin-top: 0.7rem;
}

.plans-history-list {
  margin-top: 0.8rem;
  max-height: min(52vh, 460px);
  overflow: auto;
  display: grid;
  gap: 0.72rem;
  padding-right: 0.2rem;
}

.plans-history-item {
  border: 1px solid rgba(16, 33, 42, 0.11);
  border-radius: 18px;
  padding: 0.82rem 0.9rem;
  background: rgba(255, 255, 255, 0.82);
}

.plans-history-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.6rem;
  margin-bottom: 0.55rem;
}

.plans-history-item-meta {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  flex-wrap: wrap;
  min-width: 0;
}

.plans-history-icon-btn {
  width: 2.35rem;
  min-width: 2.35rem;
  height: 2.35rem;
  border-radius: 999px;
  padding: 0;
  font-size: 1rem;
}

.plans-history-text {
  margin-bottom: 0.55rem;
}

.plans-history-files {
  margin-bottom: 0.45rem;
}

.plans-history-actions {
  margin-top: 0.8rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.plans-history-actions .btn {
  min-width: 9.5rem;
  justify-content: center;
}

.report-status-pending {
  background: rgba(201, 111, 59, 0.14);
  color: #9b4b24;
}

.report-status-completed {
  background: rgba(17, 120, 111, 0.12);
  color: #0f5e57;
}

.report-status-rejected {
  background: rgba(183, 75, 58, 0.12);
  color: #a63f32;
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

.row-edit-prorectors-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
  font-weight: 700;
}

.row-edit-prorectors-meta {
  font-size: 0.8rem;
  color: var(--muted);
  font-weight: 700;
}

.row-edit-prorectors {
  margin-top: 0.5rem;
  max-height: min(42vh, 300px);
  overflow: auto;
  padding: 0.55rem;
  border: 1px solid rgba(16, 33, 42, 0.12);
  border-radius: 18px;
  background: rgba(248, 252, 251, 0.74);
}

.prorector-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.55rem;
  margin-top: 0;
}

.prorector-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.65rem;
  align-items: start;
  padding: 0.75rem 0.82rem;
  border-radius: 14px;
  border: 1px solid rgba(16, 33, 42, 0.12);
  background: rgba(255, 255, 255, 0.92);
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease, box-shadow 0.2s ease;
}

.prorector-item:hover {
  border-color: rgba(17, 120, 111, 0.38);
  box-shadow: 0 8px 24px rgba(15, 31, 41, 0.08);
}

.prorector-item.is-selected {
  border-color: rgba(17, 120, 111, 0.52);
  background: linear-gradient(145deg, rgba(17, 120, 111, 0.12), rgba(17, 120, 111, 0.05));
}

.prorector-item input {
  width: 1.05rem;
  height: 1.05rem;
  margin-top: 0.08rem;
  accent-color: var(--accent);
}

.prorector-item span {
  display: grid;
  gap: 0.12rem;
  min-width: 0;
}

.prorector-item strong {
  font-size: 0.9rem;
  line-height: 1.35;
}

.prorector-item small {
  color: var(--muted);
  font-size: 0.78rem;
}

.row-edit-prorectors-empty {
  margin-top: 0.6rem;
}

.modal-label {
  display: grid;
  gap: 0.45rem;
  margin-top: 1rem;
}

.modal-textarea,
.report-textarea,
.report-file-input {
  width: 100%;
}

.modal-textarea {
  min-height: 8.8rem;
  resize: vertical;
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

  .plans-header-actions {
    width: 100%;
    justify-content: space-between;
  }

  .plans-card-tabs {
    max-width: 100%;
  }

  .plans-filter {
    min-width: 100%;
    max-width: none;
  }
}

@media (max-width: 980px) {
  .plans-table-card {
    padding: 0.9rem;
  }

  .plans-header-actions,
  .plans-reports-actions {
    flex-wrap: wrap;
  }

  .plans-toolbar-btn,
  .plans-export-wrap {
    width: 100%;
  }

  .plans-export-menu {
    right: 0;
    left: auto;
    min-width: 100%;
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

  .date-range-grid {
    grid-template-columns: 1fr;
  }

  .plans-edit-row-btn,
  .plans-report-btn {
    margin-top: 0.52rem;
  }

  .prorector-list {
    grid-template-columns: 1fr;
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
