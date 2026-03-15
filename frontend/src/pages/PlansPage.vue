<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import {
  fetchPlanIndicators,
  savePlanIndicator,
  submitPlanIndicatorReport
} from '../services/plan.service'
import { fetchProrectorsRequest } from '../services/user.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()
const { tr } = useLocale()

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
  initialize()
})

onBeforeUnmount(() => {
  stopCountdownTicker()
})
</script>

<template>
  <section class="plans-page">
    <PageHeader
      :title="tr('Планы', 'Жоспарлар')"
      :subtitle="tr('Рабочая матрица текущего года: мероприятия, сроки исполнения и ответственные по каждому индикатору', 'Ағымдағы жыл матрицасы: әр индикатор үшін іс-шаралар, мерзімдер және жауаптылар')"
      :eyebrow="tr('Сетка исполнения', 'Орындау кестесі')"
    />

    <div class="panel panel-strong toolbar-panel plans-toolbar">
      <div class="plans-year-card">
        <span class="kicker">{{ tr('Текущий год', 'Ағымдағы жыл') }}</span>
        <strong>{{ selectedYear }}</strong>
        <p>{{ tr('Раздел синхронизирован только с индикаторами текущего года.', 'Бұл бөлім тек ағымдағы жыл индикаторларымен синхрондалады.') }}</p>
      </div>

      <label v-if="isAdmin" class="plans-filter">
        <span>{{ tr('Ответственные', 'Жауаптылар') }}</span>
        <select v-model="selectedResponsibleFilter">
          <option value="">{{ tr('Все', 'Барлығы') }}</option>
          <option v-for="prorector in prorectors" :key="`filter-${prorector.id}`" :value="String(prorector.id)">
            {{ prorector.full_name }}
          </option>
        </select>
      </label>

      <div class="plans-visible-card">
        <span class="kicker">{{ tr('Отображено', 'Көрсетілген') }}</span>
        <strong>{{ visibleRows.length }}</strong>
      </div>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <section class="panel panel-strong plans-table-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ tr('Операционный план', 'Операциялық жоспар') }}</h3>
          <p class="panel-subtitle">{{ tr('Редактирование доступно администратору, отправка отчета - назначенному проректору.', 'Өңдеу әкімшіге қолжетімді, есеп жіберу бекітілген проректорға арналған.') }}</p>
        </div>
        <span class="kicker">{{ visibleRows.length }} {{ tr('индикаторов', 'индикатор') }}</span>
      </div>

      <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
      <div v-else-if="!hasRows" class="empty-state">
        {{ tr(`По году ${selectedYear} нет запланированных индикаторов.`, `${selectedYear} жылына жоспарланған индикатор табылмады.`) }}
      </div>
      <div v-else-if="!hasVisibleRows" class="empty-state">
        {{ tr('По выбранному ответственному индикаторы не найдены.', 'Таңдалған жауаптыға сәйкес индикатор табылмады.') }}
      </div>

      <div v-else class="table-wrapper">
        <table class="plan-table">
          <thead>
            <tr>
              <th class="col-number">№</th>
              <th>{{ tr('Индикатор Программы развития', 'Бағдарлама дамуының индикаторы') }}</th>
              <th>{{ tr('Мероприятия по достижению индикатора', 'Индикаторға жету іс-шаралары') }}</th>
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

              <td :data-label="tr('Мероприятия по достижению индикатора', 'Индикаторға жету іс-шаралары')">
                <div
                  class="table-text-preview text-pretty"
                  :class="{ 'is-empty': textPreview(row.activities) === '—' }"
                  role="button"
                  tabindex="0"
                  @click="openReadModal(tr('Мероприятия по достижению индикатора', 'Индикаторға жету іс-шаралары'), row.activities)"
                  @keyup.enter="openReadModal(tr('Мероприятия по достижению индикатора', 'Индикаторға жету іс-шаралары'), row.activities)"
                  @keyup.space.prevent="openReadModal(tr('Мероприятия по достижению индикатора', 'Индикаторға жету іс-шаралары'), row.activities)"
                >
                  <span class="table-text-preview-content">{{ textPreview(row.activities) }}</span>
                </div>
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
    </section>

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

          <label class="modal-label">
            {{ tr('Мероприятия по достижению индикатора', 'Индикаторға жету іс-шаралары') }}
            <textarea
              v-model="rowEditForm.activities"
              class="plans-edit-textarea"
              rows="8"
              :placeholder="tr('Введите мероприятия...', 'Мероприятия мәтінін жазыңыз...')"
            />
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
