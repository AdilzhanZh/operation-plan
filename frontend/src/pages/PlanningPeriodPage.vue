<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { useAuthStore } from '../store/auth'
import {
  createPlanningPeriodIndicator,
  fetchPlanningPeriod,
  importPlanningPeriodExcel,
  updatePlanningPeriodIndicator
} from '../services/planningPeriod.service'

const authStore = useAuthStore()

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
        error: 'Жыл мен мән бірге толтырылуы керек'
      }
    }

    const year = Number(rawYear)
    if (!Number.isInteger(year)) {
      return {
        error: `Жыл дұрыс емес: ${rawYear}`
      }
    }

    if (year < 2000 || year > 2100) {
      return {
        error: `Жыл диапазоны: 2000-2100 (${year})`
      }
    }

    if (payload[String(year)] !== undefined) {
      return {
        error: `Қайталанатын жыл: ${year}`
      }
    }

    payload[String(year)] = rawValue
  }

  if (Object.keys(payload).length === 0) {
    return {
      error: 'Кемінде бір жыл мен мән енгізіңіз'
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

const createForm = reactive({
  targetIndicator: '',
  unit: '',
  years: [newYearRow()]
})

const editForm = reactive({
  targetIndicator: '',
  unit: '',
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

function clearPeriodFilter() {
  periodFromYear.value = ''
  periodToYear.value = ''
}

function resetCreateForm() {
  createForm.targetIndicator = ''
  createForm.unit = ''
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
      ?? 'Мәліметтерді жүктеу мүмкін болмады'
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
    errorMessage.value = 'Excel файл таңдаңыз (.xlsx)'
    return
  }

  importing.value = true

  try {
    const response = await importPlanningPeriodExcel(importFile.value)
    successMessage.value = `Импорт аяқталды: жаңа ${response?.created ?? 0}, жаңартылды ${response?.updated ?? 0}, өткізіліп жіберілді ${response?.skipped ?? 0}`
    importFile.value = null
    importFileName.value = ''
    importInputKey.value += 1
    await loadRows()
  } catch (requestError) {
    errorMessage.value = requestError?.response?.data?.error
      ?? (typeof requestError?.response?.data === 'string' ? requestError.response.data : null)
      ?? requestError?.message
      ?? 'Импорт кезінде қате болды'
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

  if (!targetIndicator || !unit) {
    errorMessage.value = 'Целевой индикатор және ед. изм. толтырыңыз'
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
      year_values: payload
    })

    resetCreateForm()
    successMessage.value = 'Көрсеткіш сәтті қосылды'
    await loadRows()
  } catch (requestError) {
    errorMessage.value = requestError?.response?.data?.error
      ?? (typeof requestError?.response?.data === 'string' ? requestError.response.data : null)
      ?? requestError?.message
      ?? 'Қосу кезінде қате болды'
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

  if (!targetIndicator || !unit) {
    errorMessage.value = 'Целевой индикатор және ед. изм. толтырыңыз'
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
      year_values: payload
    })

    successMessage.value = 'Өзгерістер сақталды'
    editingId.value = null
    await loadRows()
  } catch (requestError) {
    errorMessage.value = requestError?.response?.data?.error
      ?? (typeof requestError?.response?.data === 'string' ? requestError.response.data : null)
      ?? requestError?.message
      ?? 'Сақтау кезінде қате болды'
  } finally {
    saving.value = false
  }
}

onMounted(loadRows)
</script>

<template>
  <section class="planning-page page">
    <PageHeader
      title="Плановый период по годам"
      subtitle="Управление целевыми индикаторами по годам, импортом Excel и диапазонной фильтрацией в едином рабочем экране"
      eyebrow="Planning"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="planning-top-grid" :class="{ 'single-column': !isAdmin }">
      <section class="panel panel-strong planning-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">Диапазон годов</h3>
            <p class="panel-subtitle">Оставьте поля пустыми, чтобы видеть весь плановый период сразу.</p>
          </div>
          <span class="kicker">Filter</span>
        </div>

        <div class="planning-filter-grid">
          <label>
            Бастапқы жыл
            <input v-model="periodFromYear" type="number" min="2000" max="2100" placeholder="2023" />
          </label>

          <label>
            Соңғы жыл
            <input v-model="periodToYear" type="number" min="2000" max="2100" placeholder="2026" />
          </label>

          <button type="button" class="btn btn-ghost planning-inline-btn" @click="clearPeriodFilter">
            Фильтрді тазалау
          </button>
        </div>
      </section>

      <section v-if="isAdmin" class="panel panel-warning planning-card">
        <div class="panel-header">
          <div>
            <h3 class="panel-title">Excel импорт</h3>
            <p class="panel-subtitle">Формат файла: `.xlsx`, обязательные колонки: индикатор, единица измерения и годы.</p>
          </div>
          <span class="kicker">Import</span>
        </div>

        <div class="planning-import-grid">
          <input
            :key="importInputKey"
            type="file"
            accept=".xlsx"
            @change="handleImportFileChange"
          />
          <button type="button" class="btn btn-accent planning-inline-btn" :disabled="importing" @click="importFromExcel">
            {{ importing ? 'Импортталуда...' : 'Импорт жасау' }}
          </button>
        </div>

        <p v-if="importFileName" class="planning-note">Таңдалған файл: {{ importFileName }}</p>
      </section>
    </div>

    <section v-if="isAdmin" class="panel panel-accent planning-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">Новый показатель</h3>
          <p class="panel-subtitle">Сначала задайте формулировку и единицу измерения, затем заполните значения по годам.</p>
        </div>
        <span class="kicker">Create</span>
      </div>

      <div class="planning-main-fields">
        <label>
          Целевой индикатор
          <textarea
            v-model="createForm.targetIndicator"
            rows="3"
            placeholder="Мысалы: Доля трудоустроенных выпускников"
          />
        </label>

        <label>
          ед. изм.
          <input
            v-model="createForm.unit"
            type="text"
            placeholder="%, место, число, балл..."
          />
        </label>
      </div>

      <div class="planning-year-editor">
        <div class="planning-year-row" v-for="yearRow in createForm.years" :key="`create-${yearRow.localId}`">
          <label>
            Жыл
            <input v-model="yearRow.year" type="number" min="2000" max="2100" placeholder="2023" />
          </label>

          <label>
            Мән
            <input v-model="yearRow.value" type="text" inputmode="decimal" placeholder="76" />
          </label>

          <button
            type="button"
            class="btn btn-danger planning-row-btn"
            @click="removeCreateYear(yearRow.localId)"
            :disabled="createForm.years.length <= 1"
          >
            Жылды өшіру
          </button>
        </div>

        <button
          v-if="canAddCreateYear"
          type="button"
          class="btn btn-ghost"
          @click="addCreateYear"
        >
          + Добавить год
        </button>
      </div>

      <div class="planning-actions">
        <button type="button" class="btn btn-primary" :disabled="creating" @click="createRow">
          {{ creating ? 'Сақталуда...' : 'Создать строку' }}
        </button>
      </div>
    </section>

    <p v-else class="message message-info">
      Бұл бөлім prorector үшін тек оқу режимінде қолжетімді.
    </p>

    <section class="panel panel-strong planning-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">Показатели планового периода</h3>
          <p class="panel-subtitle">Таблица показывает только те годы, которые попадают в текущий диапазон фильтра.</p>
        </div>
        <span class="kicker">{{ filteredRows.length }} rows</span>
      </div>

      <div v-if="loading" class="empty-state">Жүктелуде...</div>
      <template v-else>
        <div v-if="hasFilteredRows" class="table-wrap">
          <table class="table planning-table">
            <thead>
              <tr>
                <th>Целевой индикатор</th>
                <th>ед. изм.</th>
                <th v-for="year in tableYears" :key="`head-${year}`">{{ year }}</th>
                <th v-if="isAdmin">Әрекет</th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="row in filteredRows" :key="row.id">
                <td class="text-pretty">{{ row.target_indicator }}</td>
                <td>{{ row.unit }}</td>
                <td v-for="year in tableYears" :key="`${row.id}-${year}`">
                  {{ row.year_values?.[year] ?? '—' }}
                </td>
                <td v-if="isAdmin">
                  <button type="button" class="btn btn-ghost planning-table-btn" @click="startEdit(row)">
                    Өзгерту
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="hasRows" class="empty-state">
          Таңдалған период бойынша индикатор табылмады.
        </div>

        <div v-else class="empty-state">
          Кесте әзірге бос. Бірінші жолды қосқаннан кейін кесте осы жерде көрінеді.
        </div>
      </template>
    </section>

    <section v-if="isAdmin && editingId" class="panel panel-warning planning-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">Редактирование показателя</h3>
          <p class="panel-subtitle">Изменения применяются к выбранной строке и сразу обновляют общую таблицу.</p>
        </div>
        <span class="kicker">Edit</span>
      </div>

      <div class="planning-main-fields">
        <label>
          Целевой индикатор
          <textarea v-model="editForm.targetIndicator" rows="3" />
        </label>

        <label>
          ед. изм.
          <input v-model="editForm.unit" type="text" />
        </label>
      </div>

      <div class="planning-year-editor">
        <div class="planning-year-row" v-for="yearRow in editForm.years" :key="`edit-${yearRow.localId}`">
          <label>
            Жыл
            <input v-model="yearRow.year" type="number" min="2000" max="2100" />
          </label>

          <label>
            Мән
            <input v-model="yearRow.value" type="text" inputmode="decimal" />
          </label>

          <button
            type="button"
            class="btn btn-danger planning-row-btn"
            @click="removeEditYear(yearRow.localId)"
            :disabled="editForm.years.length <= 1"
          >
            Жылды өшіру
          </button>
        </div>

        <button
          v-if="canAddEditYear"
          type="button"
          class="btn btn-ghost"
          @click="addEditYear"
        >
          + Добавить год
        </button>
      </div>

      <div class="planning-actions">
        <button type="button" class="btn btn-primary" :disabled="saving" @click="saveEdit">
          {{ saving ? 'Сақталуда...' : 'Сақтау' }}
        </button>
        <button type="button" class="btn btn-ghost" @click="cancelEdit">Болдырмау</button>
      </div>
    </section>
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

.planning-note {
  margin: 0.8rem 0 0;
  color: var(--muted);
  font-size: 0.9rem;
}

.planning-main-fields {
  display: grid;
  gap: 1rem;
  grid-template-columns: minmax(0, 1.8fr) minmax(220px, 0.7fr);
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
  min-width: 860px;
}

@media (max-width: 1040px) {
  .planning-top-grid,
  .planning-main-fields,
  .planning-filter-grid,
  .planning-import-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .planning-year-row {
    grid-template-columns: 1fr;
  }
}
</style>
