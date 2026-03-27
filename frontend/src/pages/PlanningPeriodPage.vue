<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import { useAuthStore } from '../store/auth'
import {
  createPlanningPeriodIndicator,
  fetchPlanningPeriod,
  importPlanningPeriodExcel,
  updatePlanningPeriodIndicator
} from '../services/planningPeriod.service'

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
const allowedDirectionValues = directionOptions.map((item) => item.value)

let localYearRowId = 0

function newYearRow(year = '', value = '') {
  localYearRowId += 1
  return {
    localId: localYearRowId,
    year,
    value
  }
}

function isRowFilled(row) {
  return String(row.year).trim() !== '' && String(row.value).trim() !== ''
}

function rowYearNumber(row) {
  const year = Number(String(row.year).trim())
  if (!Number.isInteger(year)) {
    return null
  }

  return year
}

function normalizeYearsForEditor(yearValues) {
  const entries = Object.entries(yearValues ?? {})
    .map(([year, value]) => ({
      year,
      value: String(value)
    }))
    .sort((a, b) => Number(a.year) - Number(b.year))

  if (entries.length === 0) {
    return [newYearRow()]
  }

  return entries.map((entry) => newYearRow(entry.year, entry.value))
}

function buildYearValuesPayload(yearRows) {
  const payload = {}

  for (const row of yearRows) {
    const rawYear = String(row.year).trim()
    const rawValue = String(row.value).trim()

    if (!rawYear && !rawValue) {
      continue
    }

    if (!rawYear || !rawValue) {
      return {
        error: tr('Год и значение нужно заполнять вместе', 'Жыл мен мән бірге толтырылуы керек')
      }
    }

    const year = Number(rawYear)
    if (!Number.isInteger(year)) {
      return {
        error: tr(`Некорректный год: ${rawYear}`, `Жыл дұрыс емес: ${rawYear}`)
      }
    }

    if (year < 2000 || year > 2100) {
      return {
        error: tr(`Диапазон года: 2000-2100 (${year})`, `Жыл диапазоны: 2000-2100 (${year})`)
      }
    }

    if (payload[String(year)] !== undefined) {
      return {
        error: tr(`Повторяющийся год: ${year}`, `Қайталанатын жыл: ${year}`)
      }
    }

    payload[String(year)] = rawValue
  }

  if (Object.keys(payload).length === 0) {
    return {
      error: tr('Добавьте минимум один год и значение', 'Кемінде бір жыл мен мән енгізіңіз')
    }
  }

  return { payload }
}

function canShowAddYearButton(yearRows) {
  if (yearRows.length === 0) {
    return false
  }

  return isRowFilled(yearRows[0]) && isRowFilled(yearRows[yearRows.length - 1])
}

function pushNextYear(yearRows) {
  const lastRow = yearRows[yearRows.length - 1]
  const year = rowYearNumber(lastRow)

  if (!year) {
    return {
      error: 'Соңғы жылды дұрыс форматта енгізіңіз'
    }
  }

  yearRows.push(newYearRow(String(year + 1), ''))
  return { error: '' }
}

const rows = ref([])
const loading = ref(false)
const creating = ref(false)
const saving = ref(false)
const importing = ref(false)
const importFile = ref(null)
const importFileName = ref('')
const importInputKey = ref(0)
const editingId = ref(null)
const errorMessage = ref('')
const successMessage = ref('')
const periodFromYear = ref('')
const periodToYear = ref('')
const readModalOpen = ref(false)
const readModalTitle = ref('')
const readModalText = ref('')

const createForm = reactive({
  targetIndicator: '',
  unit: '',
  direction: '',
  years: [newYearRow()]
})

const editForm = reactive({
  targetIndicator: '',
  unit: '',
  direction: '',
  years: [newYearRow()]
})

function parseFilterYear(rawValue) {
  const normalized = String(rawValue ?? '').trim()
  if (!normalized) {
    return null
  }

  const parsed = Number(normalized)
  if (!Number.isInteger(parsed)) {
    return null
  }

  if (parsed < 2000 || parsed > 2100) {
    return null
  }

  return parsed
}

const yearRange = computed(() => {
  const from = parseFilterYear(periodFromYear.value)
  const to = parseFilterYear(periodToYear.value)

  if (from === null && to === null) {
    return { from: null, to: null }
  }
  if (from !== null && to !== null) {
    return from <= to
      ? { from, to }
      : { from: to, to: from }
  }

  return { from, to }
})

function yearInSelectedRange(yearValue) {
  const numericYear = Number(yearValue)
  if (!Number.isInteger(numericYear)) {
    return false
  }

  const { from, to } = yearRange.value
  if (from !== null && numericYear < from) {
    return false
  }
  if (to !== null && numericYear > to) {
    return false
  }
  return true
}

const filteredRows = computed(() => {
  const { from, to } = yearRange.value
  if (from === null && to === null) {
    return rows.value
  }

  return rows.value.filter((row) => Object.keys(row.year_values ?? {}).some((year) => yearInSelectedRange(year)))
})

const hasRows = computed(() => rows.value.length > 0)
const hasFilteredRows = computed(() => filteredRows.value.length > 0)
const tableYears = computed(() => {
  const allYears = new Set()

  for (const row of filteredRows.value) {
    for (const year of Object.keys(row.year_values ?? {})) {
      if (yearInSelectedRange(year)) {
        allYears.add(year)
      }
    }
  }

  return [...allYears].sort((a, b) => Number(a) - Number(b))
})

const canAddCreateYear = computed(() => canShowAddYearButton(createForm.years))
const canAddEditYear = computed(() => canShowAddYearButton(editForm.years))
const isAdmin = computed(() => authStore.user?.role === 'admin')

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

function textPreview(value) {
  const normalized = String(value ?? '').trim()
  return normalized || '—'
}

function directionLabel(value) {
  const normalized = String(value ?? '').trim()
  if (!normalized) {
    return '—'
  }

  const option = directionOptions.find((item) => item.value === normalized)
  if (!option) {
    return normalized
  }

  return tr(option.ru, option.kz)
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

function clearPeriodFilter() {
  periodFromYear.value = ''
  periodToYear.value = ''
}

function resetCreateForm() {
  createForm.targetIndicator = ''
  createForm.unit = ''
  createForm.direction = ''
  createForm.years.splice(0, createForm.years.length, newYearRow())
}

function startEdit(row) {
  if (!isAdmin.value) {
    return
  }

  clearMessages()
  editingId.value = row.id
  editForm.targetIndicator = row.target_indicator
  editForm.unit = row.unit
  editForm.direction = row.direction || ''

  const years = normalizeYearsForEditor(row.year_values)
  editForm.years.splice(0, editForm.years.length, ...years)
}

function cancelEdit() {
  editingId.value = null
  clearMessages()
}

async function loadRows() {
  loading.value = true
  errorMessage.value = ''

  try {
    const response = await fetchPlanningPeriod()
    rows.value = response.items ?? []
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось загрузить данные', 'Мәліметтерді жүктеу мүмкін болмады')
  } finally {
    loading.value = false
  }
}

function addCreateYear() {
  if (!isAdmin.value) {
    return
  }

  const { error } = pushNextYear(createForm.years)
  if (error) {
    errorMessage.value = error
  }
}

function addEditYear() {
  if (!isAdmin.value) {
    return
  }

  const { error } = pushNextYear(editForm.years)
  if (error) {
    errorMessage.value = error
  }
}

function handleImportFileChange(event) {
  const files = event?.target?.files ?? []
  const file = files[0]

  importFile.value = file ?? null
  importFileName.value = file?.name ?? ''
}

async function importFromExcel() {
  if (!isAdmin.value) {
    return
  }

  clearMessages()

  if (!importFile.value) {
    errorMessage.value = tr('Выберите Excel файл (.xlsx)', 'Excel файл таңдаңыз (.xlsx)')
    return
  }

  importing.value = true

  try {
    const response = await importPlanningPeriodExcel(importFile.value)
    successMessage.value = tr(
      `Импорт завершен: новых ${response?.created ?? 0}, обновлено ${response?.updated ?? 0}, пропущено ${response?.skipped ?? 0}`,
      `Импорт аяқталды: жаңа ${response?.created ?? 0}, жаңартылды ${response?.updated ?? 0}, өткізіліп жіберілді ${response?.skipped ?? 0}`
    )
    importFile.value = null
    importFileName.value = ''
    importInputKey.value += 1
    await loadRows()
  } catch (requestError) {
    errorMessage.value = requestError?.response?.data?.error
      ?? (typeof requestError?.response?.data === 'string' ? requestError.response.data : null)
      ?? requestError?.message
      ?? tr('Ошибка при импорте', 'Импорт кезінде қате болды')
  } finally {
    importing.value = false
  }
}

function removeCreateYear(localId) {
  if (!isAdmin.value) {
    return
  }

  if (createForm.years.length <= 1) {
    return
  }

  const next = createForm.years.filter((row) => row.localId !== localId)
  createForm.years.splice(0, createForm.years.length, ...next)
}

function removeEditYear(localId) {
  if (!isAdmin.value) {
    return
  }

  if (editForm.years.length <= 1) {
    return
  }

  const next = editForm.years.filter((row) => row.localId !== localId)
  editForm.years.splice(0, editForm.years.length, ...next)
}

async function createRow() {
  if (!isAdmin.value) {
    return
  }

  clearMessages()

  const targetIndicator = createForm.targetIndicator.trim()
  const unit = createForm.unit.trim()
  const direction = createForm.direction.trim()

  if (!targetIndicator || !unit || !direction) {
    errorMessage.value = tr('Заполните целевой индикатор, ед. изм. и направление.', 'Целевой индикатор, ед. изм. және бағытты толтырыңыз')
    return
  }
  if (!allowedDirectionValues.includes(direction)) {
    errorMessage.value = tr('Выберите корректное направление.', 'Дұрыс бағытты таңдаңыз.')
    return
  }

  const { payload, error } = buildYearValuesPayload(createForm.years)
  if (error) {
    errorMessage.value = error
    return
  }

  creating.value = true

  try {
    await createPlanningPeriodIndicator({
      target_indicator: targetIndicator,
      unit,
      direction,
      year_values: payload
    })

    resetCreateForm()
    successMessage.value = tr('Показатель успешно добавлен', 'Көрсеткіш сәтті қосылды')
    await loadRows()
  } catch (requestError) {
    errorMessage.value = requestError?.response?.data?.error
      ?? (typeof requestError?.response?.data === 'string' ? requestError.response.data : null)
      ?? requestError?.message
      ?? tr('Ошибка при добавлении', 'Қосу кезінде қате болды')
  } finally {
    creating.value = false
  }
}

async function saveEdit() {
  if (!isAdmin.value) {
    return
  }

  if (!editingId.value) {
    return
  }

  clearMessages()

  const targetIndicator = editForm.targetIndicator.trim()
  const unit = editForm.unit.trim()
  const direction = editForm.direction.trim()

  if (!targetIndicator || !unit || !direction) {
    errorMessage.value = tr('Заполните целевой индикатор, ед. изм. и направление.', 'Целевой индикатор, ед. изм. және бағытты толтырыңыз')
    return
  }
  if (!allowedDirectionValues.includes(direction)) {
    errorMessage.value = tr('Выберите корректное направление.', 'Дұрыс бағытты таңдаңыз.')
    return
  }

  const { payload, error } = buildYearValuesPayload(editForm.years)
  if (error) {
    errorMessage.value = error
    return
  }

  saving.value = true

  try {
    await updatePlanningPeriodIndicator(editingId.value, {
      target_indicator: targetIndicator,
      unit,
      direction,
      year_values: payload
    })

    successMessage.value = tr('Изменения сохранены', 'Өзгерістер сақталды')
    editingId.value = null
    await loadRows()
  } catch (requestError) {
    errorMessage.value = requestError?.response?.data?.error
      ?? (typeof requestError?.response?.data === 'string' ? requestError.response.data : null)
      ?? requestError?.message
      ?? tr('Ошибка при сохранении', 'Сақтау кезінде қате болды')
  } finally {
    saving.value = false
  }
}

onMounted(loadRows)
</script>

<template>
  <section class="planning-page page">
    <PageHeader
      :title="tr('Плановый период по годам', 'Жылдар бойынша жоспарлы кезең')"
      :subtitle="tr('Управление целевыми индикаторами по годам, импортом Excel и диапазонной фильтрацией в едином рабочем экране', 'Жылдар бойынша мақсатты индикаторларды, Excel импортын және диапазон сүзгісін бір экранда басқару')"
      :eyebrow="tr('Планирование', 'Жоспарлау')"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="planning-top-grid" :class="{ 'single-column': !isAdmin }">
      <section class="panel panel-strong planning-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Диапазон годов', 'Жылдар диапазоны') }}</h3>
            <p class="panel-subtitle">{{ tr('Оставьте поля пустыми, чтобы видеть весь плановый период сразу.', 'Барлық жоспарлы кезеңді көру үшін өрістерді бос қалдырыңыз.') }}</p>
          </div>
          <span class="kicker">{{ tr('Фильтр', 'Сүзгі') }}</span>
        </div>

        <div class="planning-filter-grid">
          <label>
            {{ tr('Начальный год', 'Бастапқы жыл') }}
            <input v-model="periodFromYear" type="number" min="2000" max="2100" placeholder="2023" />
          </label>

          <label>
            {{ tr('Конечный год', 'Соңғы жыл') }}
            <input v-model="periodToYear" type="number" min="2000" max="2100" placeholder="2026" />
          </label>

          <button type="button" class="btn btn-ghost planning-inline-btn" @click="clearPeriodFilter">
            {{ tr('Очистить фильтр', 'Фильтрді тазалау') }}
          </button>
        </div>
      </section>

      <section v-if="isAdmin" class="panel panel-warning planning-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">{{ tr('Excel импорт', 'Excel импорт') }}</h3>
            <p class="panel-subtitle">{{ tr('Формат файла: `.xlsx`, обязательные колонки: индикатор, единица измерения и годы.', 'Файл форматы: `.xlsx`, міндетті бағандар: индикатор, өлшем бірлігі және жылдар.') }}</p>
          </div>
          <span class="kicker">{{ tr('Импорт', 'Импорт') }}</span>
        </div>

        <div class="planning-import-grid">
          <div class="planning-import-file">
            <input
              :id="`planning-import-file-${importInputKey}`"
              :key="importInputKey"
              class="planning-file-input"
              type="file"
              accept=".xlsx"
              @change="handleImportFileChange"
            />
            <label
              :for="`planning-import-file-${importInputKey}`"
              class="btn btn-ghost planning-file-trigger"
            >
              {{ tr('Выбрать файл', 'Файлды таңдау') }}
            </label>
            <span class="planning-file-name" :class="{ 'is-placeholder': !importFileName }">
              {{ importFileName || tr('Файл не выбран', 'Файл таңдалмаған') }}
            </span>
          </div>
          <button type="button" class="btn btn-accent planning-inline-btn" :disabled="importing" @click="importFromExcel">
            {{ importing ? tr('Импорт...', 'Импортталуда...') : tr('Импортировать', 'Импорт жасау') }}
          </button>
        </div>
      </section>
    </div>

    <section v-if="isAdmin" class="panel panel-accent planning-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ tr('Новый показатель', 'Жаңа көрсеткіш') }}</h3>
          <p class="panel-subtitle">{{ tr('Сначала задайте формулировку и единицу измерения, затем заполните значения по годам.', 'Алдымен тұжырым мен өлшем бірлігін беріңіз, содан кейін жылдар бойынша мәндерді толтырыңыз.') }}</p>
        </div>
        <span class="kicker">{{ tr('Создать', 'Құру') }}</span>
      </div>

      <div class="planning-main-fields">
        <label>
          {{ tr('Целевой индикатор', 'Мақсатты индикатор') }}
          <textarea
            v-model="createForm.targetIndicator"
            rows="3"
            :placeholder="tr('Например: Доля трудоустроенных выпускников', 'Мысалы: Доля трудоустроенных выпускников')"
          />
        </label>

        <label>
          {{ tr('ед. изм.', 'өлш. бірл.') }}
          <input
            v-model="createForm.unit"
            type="text"
            placeholder="%, место, число, балл..."
          />
        </label>

        <label>
          {{ tr('Направление', 'Бағыт') }}
          <select v-model="createForm.direction">
            <option value="" disabled>{{ tr('Выберите направление', 'Бағытты таңдаңыз') }}</option>
            <option v-for="option in directionOptions" :key="`create-direction-${option.value}`" :value="option.value">
              {{ tr(option.ru, option.kz) }}
            </option>
          </select>
        </label>
      </div>

      <div class="planning-year-editor">
        <div class="planning-year-row" v-for="yearRow in createForm.years" :key="`create-${yearRow.localId}`">
          <label>
            {{ tr('Год', 'Жыл') }}
            <input v-model="yearRow.year" type="number" min="2000" max="2100" placeholder="2023" />
          </label>

          <label>
            {{ tr('Значение', 'Мән') }}
            <input v-model="yearRow.value" type="text" inputmode="decimal" placeholder="76" />
          </label>

          <button
            type="button"
            class="btn btn-danger planning-row-btn"
            @click="removeCreateYear(yearRow.localId)"
            :disabled="createForm.years.length <= 1"
          >
            {{ tr('Удалить год', 'Жылды өшіру') }}
          </button>
        </div>

        <button
          v-if="canAddCreateYear"
          type="button"
          class="btn btn-ghost"
          @click="addCreateYear"
        >
          {{ tr('+ Добавить год', '+ Жыл қосу') }}
        </button>
      </div>

      <div class="planning-actions">
        <button type="button" class="btn btn-primary" :disabled="creating" @click="createRow">
          {{ creating ? tr('Сохранение...', 'Сақталуда...') : tr('Создать строку', 'Жол құру') }}
        </button>
      </div>
    </section>

    <p v-else class="message message-info">
      {{ tr('Этот раздел для сотрудника доступен только в режиме просмотра.', 'Бұл бөлім қызметкер үшін тек оқу режимінде қолжетімді.') }}
    </p>

    <section class="panel panel-strong planning-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ tr('Показатели планового периода', 'Жоспарлы кезең көрсеткіштері') }}</h3>
          <p class="panel-subtitle">{{ tr('Таблица показывает только те годы, которые попадают в текущий диапазон фильтра.', 'Кесте ағымдағы сүзгі диапазонына кіретін жылдарды ғана көрсетеді.') }}</p>
        </div>
        <span class="kicker">{{ filteredRows.length }} {{ tr('строк', 'жол') }}</span>
      </div>

      <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
      <template v-else>
        <div v-if="hasFilteredRows" class="table-wrap planning-table-wrap">
          <table class="table planning-table">
            <thead>
              <tr>
                <th class="col-sticky-indicator">{{ tr('Целевой индикатор', 'Мақсатты индикатор') }}</th>
                <th class="col-sticky-unit">{{ tr('ед. изм.', 'өлш. бірл.') }}</th>
                <th class="col-direction">{{ tr('Направление', 'Бағыт') }}</th>
                <th v-for="year in tableYears" :key="`head-${year}`">{{ year }}</th>
                <th v-if="isAdmin">{{ tr('Действие', 'Әрекет') }}</th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="row in filteredRows" :key="row.id">
                <td class="col-sticky-indicator">
                  <div
                    class="table-text-preview text-pretty"
                    :class="{ 'is-empty': textPreview(row.target_indicator) === '—' }"
                    role="button"
                    tabindex="0"
                    @click="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.target_indicator)"
                    @keyup.enter="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.target_indicator)"
                    @keyup.space.prevent="openReadModal(tr('Целевой индикатор', 'Мақсатты индикатор'), row.target_indicator)"
                  >
                    <span class="table-text-preview-content">{{ textPreview(row.target_indicator) }}</span>
                  </div>
                </td>
                <td class="col-sticky-unit">{{ row.unit }}</td>
                <td class="col-direction">
                  <div
                    class="table-text-preview text-pretty"
                    :class="{ 'is-empty': directionLabel(row.direction) === '—' }"
                    role="button"
                    tabindex="0"
                    @click="openReadModal(tr('Направление', 'Бағыт'), directionLabel(row.direction))"
                    @keyup.enter="openReadModal(tr('Направление', 'Бағыт'), directionLabel(row.direction))"
                    @keyup.space.prevent="openReadModal(tr('Направление', 'Бағыт'), directionLabel(row.direction))"
                  >
                    <span class="table-text-preview-content">{{ directionLabel(row.direction) }}</span>
                  </div>
                </td>
                <td v-for="year in tableYears" :key="`${row.id}-${year}`">
                  {{ row.year_values?.[year] ?? '—' }}
                </td>
                <td v-if="isAdmin">
                  <button type="button" class="btn btn-ghost planning-table-btn" @click="startEdit(row)">
                    {{ tr('Изменить', 'Өзгерту') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="hasRows" class="empty-state">
          {{ tr('По выбранному периоду индикаторы не найдены.', 'Таңдалған период бойынша индикатор табылмады.') }}
        </div>

        <div v-else class="empty-state">
          {{ tr('Таблица пока пустая. После добавления первой строки она появится здесь.', 'Кесте әзірге бос. Бірінші жолды қосқаннан кейін кесте осы жерде көрінеді.') }}
        </div>
      </template>
    </section>

    <div v-if="readModalOpen" class="modal-backdrop" @click.self="closeReadModal">
      <div class="modal-card planning-read-modal">
        <h3 class="modal-title">{{ readModalTitle }}</h3>
        <div class="planning-read-content text-pretty">
          {{ readModalText }}
        </div>
        <div class="modal-actions">
          <button type="button" class="btn btn-primary" @click="closeReadModal">{{ tr('Закрыть', 'Жабу') }}</button>
        </div>
      </div>
    </div>

    <div v-if="isAdmin && editingId" class="modal-backdrop" @click.self="cancelEdit">
      <div class="modal-card planning-edit-modal">
        <h3 class="modal-title">{{ tr('Редактирование показателя', 'Көрсеткішті өзгерту') }}</h3>
        <p class="modal-subtitle">{{ tr('Изменения сохраняются в общей таблице сразу после подтверждения.', 'Өзгерістер растаудан кейін кестеге бірден сақталады.') }}</p>
        <p v-if="errorMessage" class="message message-error modal-feedback">{{ errorMessage }}</p>
        <p v-if="successMessage" class="message message-success modal-feedback">{{ successMessage }}</p>

        <div class="planning-main-fields">
          <label>
            {{ tr('Целевой индикатор', 'Мақсатты индикатор') }}
            <textarea v-model="editForm.targetIndicator" rows="3" />
          </label>

          <label>
            {{ tr('ед. изм.', 'өлш. бірл.') }}
            <input v-model="editForm.unit" type="text" />
          </label>

          <label>
            {{ tr('Направление', 'Бағыт') }}
            <select v-model="editForm.direction">
              <option value="" disabled>{{ tr('Выберите направление', 'Бағытты таңдаңыз') }}</option>
              <option v-for="option in directionOptions" :key="`edit-direction-${option.value}`" :value="option.value">
                {{ tr(option.ru, option.kz) }}
              </option>
            </select>
          </label>
        </div>

        <div class="planning-year-editor planning-year-editor-modal">
          <div class="planning-year-row" v-for="yearRow in editForm.years" :key="`edit-${yearRow.localId}`">
            <label>
              {{ tr('Год', 'Жыл') }}
              <input v-model="yearRow.year" type="number" min="2000" max="2100" />
            </label>

            <label>
              {{ tr('Значение', 'Мән') }}
              <input v-model="yearRow.value" type="text" inputmode="decimal" />
            </label>

            <button
              type="button"
              class="btn btn-danger planning-row-btn"
              @click="removeEditYear(yearRow.localId)"
              :disabled="editForm.years.length <= 1"
            >
              {{ tr('Удалить год', 'Жылды өшіру') }}
            </button>
          </div>

          <button
            v-if="canAddEditYear"
            type="button"
            class="btn btn-ghost"
            @click="addEditYear"
          >
            {{ tr('+ Добавить год', '+ Жыл қосу') }}
          </button>
        </div>

        <div class="modal-actions">
          <button type="button" class="btn btn-ghost" @click="cancelEdit">{{ tr('Отмена', 'Болдырмау') }}</button>
          <button type="button" class="btn btn-primary" :disabled="saving" @click="saveEdit">
            {{ saving ? tr('Сохранение...', 'Сақталуда...') : tr('Сохранить', 'Сақтау') }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.planning-top-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 0.9fr);
}

.planning-top-grid.single-column {
  grid-template-columns: 1fr;
}

.planning-card {
  padding: 1.2rem;
}

.planning-filter-grid,
.planning-import-grid {
  display: grid;
  gap: 0.9rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  align-items: end;
}

.planning-inline-btn {
  justify-content: center;
}

.planning-import-file {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
}

.planning-file-input {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.planning-file-trigger {
  min-width: 11rem;
  justify-content: center;
}

.planning-file-name {
  min-width: 0;
  min-height: 2.85rem;
  display: flex;
  align-items: center;
  padding: 0.78rem 0.92rem;
  border-radius: 16px;
  border: 1px solid rgba(16, 33, 42, 0.12);
  background: rgba(255, 255, 255, 0.86);
  color: var(--text);
  line-height: 1.35;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.planning-file-name.is-placeholder {
  color: var(--muted);
}

.planning-note {
  margin: 0.8rem 0 0;
  color: var(--muted);
  font-size: 0.9rem;
}

.planning-main-fields {
  display: grid;
  gap: 1rem;
  grid-template-columns: minmax(0, 1.6fr) minmax(220px, 0.6fr) minmax(260px, 0.9fr);
}

.planning-year-editor {
  margin-top: 1rem;
  display: grid;
  gap: 0.85rem;
}

.planning-year-row {
  display: grid;
  gap: 0.85rem;
  grid-template-columns: minmax(120px, 0.65fr) minmax(160px, 0.85fr) auto;
  align-items: end;
}

.planning-row-btn,
.planning-table-btn {
  justify-content: center;
}

.planning-actions {
  margin-top: 1rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.planning-table {
  min-width: 1280px;
}

.planning-table-wrap {
  overflow-x: auto;
}

.planning-table th,
.planning-table td {
  position: relative;
}

.planning-table .col-sticky-indicator {
  position: sticky;
  left: 0;
  min-width: 360px;
  width: 360px;
  z-index: 5;
}

.planning-table .col-sticky-unit {
  position: sticky;
  left: 360px;
  min-width: 110px;
  width: 110px;
  z-index: 6;
}

.planning-table .col-direction {
  min-width: 320px;
  width: 320px;
}

.planning-table thead .col-sticky-indicator,
.planning-table thead .col-sticky-unit {
  z-index: 7;
  background: #f5f0e8;
}

.planning-table tbody tr {
  --planning-sticky-bg: #fffdf9;
}

.planning-table tbody tr:nth-child(even) {
  --planning-sticky-bg: #f8f3ec;
}

.planning-table tbody tr:hover {
  --planning-sticky-bg: #e7f3f1;
}

.planning-table tbody td.col-sticky-indicator,
.planning-table tbody td.col-sticky-unit {
  background: var(--planning-sticky-bg);
  background-clip: padding-box;
}

.planning-table .col-sticky-unit::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  right: -1px;
  width: 1px;
  background: rgba(16, 33, 42, 0.14);
}

.planning-read-modal {
  width: min(760px, 100%);
}

.planning-edit-modal {
  width: min(960px, 100%);
}

.planning-year-editor-modal {
  max-height: min(42vh, 360px);
  overflow: auto;
  padding-right: 0.2rem;
}

.planning-read-content {
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

@media (max-width: 1040px) {
  .planning-top-grid,
  .planning-main-fields,
  .planning-filter-grid,
  .planning-import-grid {
    grid-template-columns: 1fr;
  }

  .planning-import-file {
    grid-template-columns: 1fr;
  }

  .planning-edit-modal {
    width: min(760px, 100%);
  }
}

@media (max-width: 760px) {
  .planning-year-row {
    grid-template-columns: 1fr;
  }
}
</style>
