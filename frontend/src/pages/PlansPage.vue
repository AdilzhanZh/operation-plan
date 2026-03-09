<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { fetchPlanIndicators, fetchPlanYears, savePlanIndicator } from '../services/plan.service'
import { useAuthStore } from '../store/auth'

const authStore = useAuthStore()

const years = ref([])
const selectedYear = ref('')
const rows = ref([])
const loading = ref(false)
const savingIndicatorId = ref(null)
const errorMessage = ref('')
const successMessage = ref('')

const isAdmin = computed(() => authStore.user?.role === 'admin')
const hasYears = computed(() => years.value.length > 0)
const canLoadYear = computed(() => selectedYear.value !== '')

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
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
      responsible: row.responsible
    })

    Object.assign(row, saved)
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
              </template>
              <template v-else>
                <div class="cell-readonly">{{ row.development_indicator }}</div>
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
                <input v-model="row.responsible" class="cell-input" type="text" />
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
              </template>
            </td>
          </tr>
        </tbody>
      </table>
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

.cell-readonly {
  white-space: pre-wrap;
  color: #0f172a;
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

@media (max-width: 900px) {
  .toolbar {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.55rem;
  }
}
</style>
