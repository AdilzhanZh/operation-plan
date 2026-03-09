<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import {
  createPlanningPeriodIndicator,
  fetchPlanningPeriod,
  updatePlanningPeriodIndicator
} from '../services/planningPeriod.service'

let localYearRowId = 0

function newYearRow(year = '', value = '') {
  localYearRowId += 1
  return {
    localId: localYearRowId,
    year,
    value
  }
}

function parseNumber(value) {
  if (value === undefined || value === null) {
    return null
  }

  const normalized = String(value).trim().replace(',', '.')
  if (!normalized) {
    return null
  }

  const parsed = Number(normalized)
  if (Number.isNaN(parsed)) {
    return Number.NaN
  }

  return parsed
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

    const value = parseNumber(rawValue)
    if (value === null || Number.isNaN(value)) {
      return {
        error: `Мән дұрыс емес (${year})`
      }
    }

    if (payload[String(year)] !== undefined) {
      return {
        error: `Қайталанатын жыл: ${year}`
      }
    }

    payload[String(year)] = value
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
const editingId = ref(null)
const errorMessage = ref('')
const successMessage = ref('')

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

const hasRows = computed(() => rows.value.length > 0)
const tableYears = computed(() => {
  const allYears = new Set()

  for (const row of rows.value) {
    for (const year of Object.keys(row.year_values ?? {})) {
      allYears.add(year)
    }
  }

  return [...allYears].sort((a, b) => Number(a) - Number(b))
})

const canAddCreateYear = computed(() => canShowAddYearButton(createForm.years))
const canAddEditYear = computed(() => canShowAddYearButton(editForm.years))

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

function resetCreateForm() {
  createForm.targetIndicator = ''
  createForm.unit = ''
  createForm.years.splice(0, createForm.years.length, newYearRow())
}

function startEdit(row) {
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
  const { error } = pushNextYear(createForm.years)
  if (error) {
    errorMessage.value = error
  }
}

function addEditYear() {
  const { error } = pushNextYear(editForm.years)
  if (error) {
    errorMessage.value = error
  }
}

function removeCreateYear(localId) {
  if (createForm.years.length <= 1) {
    return
  }

  const next = createForm.years.filter((row) => row.localId !== localId)
  createForm.years.splice(0, createForm.years.length, ...next)
}

function removeEditYear(localId) {
  if (editForm.years.length <= 1) {
    return
  }

  const next = editForm.years.filter((row) => row.localId !== localId)
  editForm.years.splice(0, editForm.years.length, ...next)
}

async function createRow() {
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
  <section>
    <PageHeader
      title="Плановый период по годам"
      subtitle="Алдымен көрсеткішті жасаңыз, содан кейін кесте автоматты түрде пайда болады"
    />

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>
    <p v-if="successMessage" class="message message-success">{{ successMessage }}</p>

    <div class="card">
      <h3>Добавить показатель</h3>

      <div class="main-fields">
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

      <div class="year-editor">
        <div class="year-row" v-for="yearRow in createForm.years" :key="`create-${yearRow.localId}`">
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
            class="ghost danger"
            @click="removeCreateYear(yearRow.localId)"
            :disabled="createForm.years.length <= 1"
          >
            Жылды өшіру
          </button>
        </div>

        <button
          v-if="canAddCreateYear"
          type="button"
          class="ghost"
          @click="addCreateYear"
        >
          + Добавить год
        </button>
      </div>

      <button type="button" class="primary" :disabled="creating" @click="createRow">
        {{ creating ? 'Сақталуда...' : 'Создать строку' }}
      </button>
    </div>

    <div v-if="loading" class="loading">Жүктелуде...</div>

    <template v-else>
      <div v-if="hasRows" class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Целевой индикатор</th>
              <th>ед. изм.</th>
              <th v-for="year in tableYears" :key="`head-${year}`">{{ year }}</th>
              <th>Әрекет</th>
            </tr>
          </thead>

          <tbody>
            <tr v-for="row in rows" :key="row.id">
              <td>{{ row.target_indicator }}</td>
              <td>{{ row.unit }}</td>
              <td v-for="year in tableYears" :key="`${row.id}-${year}`">
                {{ row.year_values?.[year] ?? '—' }}
              </td>
              <td>
                <button type="button" class="ghost" @click="startEdit(row)">Өзгерту</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <p v-else class="empty-note">
        Кесте әзірге бос. Бірінші жолды қосқаннан кейін кесте осы жерде көрінеді.
      </p>
    </template>

    <div v-if="editingId" class="card edit-card">
      <h3>Көрсеткішті өзгерту</h3>

      <div class="main-fields">
        <label>
          Целевой индикатор
          <textarea v-model="editForm.targetIndicator" rows="3" />
        </label>

        <label>
          ед. изм.
          <input v-model="editForm.unit" type="text" />
        </label>
      </div>

      <div class="year-editor">
        <div class="year-row" v-for="yearRow in editForm.years" :key="`edit-${yearRow.localId}`">
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
            class="ghost danger"
            @click="removeEditYear(yearRow.localId)"
            :disabled="editForm.years.length <= 1"
          >
            Жылды өшіру
          </button>
        </div>

        <button
          v-if="canAddEditYear"
          type="button"
          class="ghost"
          @click="addEditYear"
        >
          + Добавить год
        </button>
      </div>

      <div class="edit-actions">
        <button type="button" class="primary" :disabled="saving" @click="saveEdit">
          {{ saving ? 'Сақталуда...' : 'Сақтау' }}
        </button>
        <button type="button" class="ghost" @click="cancelEdit">Болдырмау</button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.card {
  margin-bottom: 1rem;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 1rem;
}

h3 {
  margin: 0 0 0.9rem;
}

.message {
  margin: 0.5rem 0 1rem;
  padding: 0.7rem 0.9rem;
  border-radius: 8px;
  font-size: 0.92rem;
}

.message-error {
  background: #fee2e2;
  color: #991b1b;
  border: 1px solid #fecaca;
}

.message-success {
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
}

.main-fields {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 0.9rem;
}

label {
  display: grid;
  gap: 0.35rem;
  font-size: 0.88rem;
}

input,
textarea,
button {
  font: inherit;
}

input,
textarea {
  width: 100%;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  padding: 0.5rem 0.6rem;
}

.year-editor {
  margin-top: 0.9rem;
  display: grid;
  gap: 0.7rem;
}

.year-row {
  display: grid;
  gap: 0.7rem;
  grid-template-columns: minmax(120px, 1fr) minmax(140px, 1fr) auto;
  align-items: end;
}

.primary,
.ghost {
  border-radius: 8px;
  border: 1px solid transparent;
  padding: 0.45rem 0.8rem;
  font-weight: 600;
  cursor: pointer;
}

.primary {
  margin-top: 0.9rem;
  background: #0f172a;
  color: #f8fafc;
}

.ghost {
  background: #ffffff;
  color: #0f172a;
  border-color: #cbd5e1;
}

.danger {
  color: #991b1b;
  border-color: #fecaca;
  background: #fff5f5;
}

.primary:disabled,
.ghost:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.table-wrap {
  overflow-x: auto;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  margin-bottom: 1rem;
}

.table {
  width: 100%;
  min-width: 820px;
  border-collapse: collapse;
}

th,
td {
  padding: 0.65rem 0.7rem;
  border-bottom: 1px solid #f1f5f9;
  text-align: left;
  vertical-align: top;
}

th {
  background: #f8fafc;
  color: #475569;
}

.loading,
.empty-note {
  margin: 0.4rem 0 1rem;
  padding: 0.8rem;
  border-radius: 8px;
  background: #f8fafc;
  color: #334155;
}

.edit-card {
  border-color: #bfdbfe;
  background: #f8fbff;
}

.edit-actions {
  margin-top: 0.9rem;
  display: flex;
  gap: 0.6rem;
}

@media (max-width: 980px) {
  .main-fields {
    grid-template-columns: 1fr;
  }

  .year-row {
    grid-template-columns: 1fr;
  }
}
</style>
