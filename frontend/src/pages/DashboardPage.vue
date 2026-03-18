<script setup>
import { computed, onMounted, ref } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { useLocale } from '../composables/useLocale'
import { fetchPlanIndicators, fetchPlanYears } from '../services/plan.service'
import { useAuthStore } from '../store/auth'

const years = ref([])
const selectedYear = ref('')
const loading = ref(false)
const errorMessage = ref('')
const allRows = ref([])
const authStore = useAuthStore()
const { tr } = useLocale()
const isProrector = computed(() => authStore.user?.role === 'prorector')
const readModalOpen = ref(false)
const readModalTitle = ref('')
const readModalText = ref('')

const activeCard = ref('total')
const stats = ref({
  total: 0,
  completed: 0,
  pending: 0,
  not_filled: 0,
  in_progress: 0,
  overdue: 0
})

const hasYears = computed(() => years.value.length > 0)

const cardConfig = computed(() => {
  if (isProrector.value) {
    return [
      { key: 'total', label: tr('Всего задач', 'Барлық тапсырма') },
      { key: 'pending', label: tr('На проверке', 'Тексерісте') },
      { key: 'completed', label: tr('Принято', 'Қабылданды') },
      { key: 'overdue', label: tr('Просрочено', 'Мерзімі өткен') }
    ]
  }

  return [
    { key: 'total', label: tr('Всего задач', 'Барлық тапсырма') },
    { key: 'completed', label: tr('Принято', 'Қабылданды') },
    { key: 'pending', label: tr('На проверке', 'Тексерісте') },
    { key: 'not_filled', label: tr('Не заполнено', 'Толтырылмаған') },
    { key: 'in_progress', label: tr('В работе', 'Жұмыста') },
    { key: 'overdue', label: tr('Просрочено', 'Мерзімі өткен') }
  ]
})

function cardMetaLabel(cardKey) {
  if (cardKey === 'total') return tr('Полный пул индикаторов выбранного года', 'Таңдалған жылдың толық индикаторлар пулы')
  if (cardKey === 'completed') return tr('Позиции с принятым результатом', 'Қабылданған нәтижесі бар позициялар')
  if (cardKey === 'pending') return tr('Индикаторы, ожидающие проверки администратора', 'Әкімші тексеруін күтіп тұрған индикаторлар')
  if (cardKey === 'not_filled') return tr('Индикаторы без заполненного графика или отчета', 'Кестесі не есебі толтырылмаған индикаторлар')
  if (cardKey === 'in_progress') return tr('Активные задачи в пределах срока', 'Мерзім ішіндегі белсенді тапсырмалар')
  if (cardKey === 'overdue') return tr('Точки, требующие немедленного контроля', 'Жедел бақылауды қажет ететін позициялар')
  return ''
}

const cards = computed(() => cardConfig.value.map((card) => ({
  ...card,
  value: stats.value[card.key] ?? 0,
  meta: cardMetaLabel(card.key)
})))

const activeCardDetails = computed(() => cards.value.find((card) => card.key === activeCard.value) ?? cards.value[0] ?? null)

const completionRateText = computed(() => {
  if (!stats.value.total) {
    return '0%'
  }

  return `${Math.round((stats.value.completed / stats.value.total) * 100)}%`
})

const currentUserFirstName = computed(() => {
  const explicitName = String(authStore.user?.first_name ?? '').trim()
  if (explicitName) {
    return explicitName
  }

  const fullName = String(authStore.user?.full_name ?? '').trim()
  if (fullName) {
    return fullName.split(/\s+/)[0]
  }

  return tr('коллега', 'әріптес')
})

const dashboardSeries = computed(() => {
  const series = [
    {
      key: 'completed',
      label: tr('Принято', 'Қабылданды'),
      value: stats.value.completed,
      tone: 'is-success',
      caption: tr('Подтверждённые результаты', 'Расталған нәтижелер')
    },
    {
      key: 'pending',
      label: tr('На проверке', 'Тексерісте'),
      value: stats.value.pending,
      tone: 'is-accent',
      caption: tr('Ожидают решения', 'Шешімді күтуде')
    },
    {
      key: 'in_progress',
      label: tr('В работе', 'Жұмыста'),
      value: stats.value.in_progress,
      tone: 'is-warning',
      caption: tr('Активный пул задач', 'Белсенді тапсырмалар')
    },
    {
      key: 'overdue',
      label: tr('Просрочено', 'Мерзімі өткен'),
      value: stats.value.overdue,
      tone: 'is-danger',
      caption: tr('Требуют вмешательства', 'Жедел араласуды қажет етеді')
    },
    {
      key: 'not_filled',
      label: tr('Не заполнено', 'Толтырылмаған'),
      value: stats.value.not_filled,
      tone: 'is-muted',
      caption: tr('Нет отчёта или срока', 'Есеп не мерзім жоқ')
    }
  ]

  const maxValue = Math.max(...series.map((item) => item.value), 1)

  return series.map((item) => ({
    ...item,
    dotCount: item.value === 0 ? 1 : Math.max(2, Math.min(12, Math.round((item.value / maxValue) * 12)))
  }))
})

const priorityRows = computed(() => {
  const statusRank = {
    overdue: 0,
    pending: 1,
    in_progress: 2,
    not_filled: 3,
    completed: 4
  }

  return [...allRows.value]
    .sort((left, right) => {
      const leftStatus = String(left.dashboard_status ?? '').toLowerCase()
      const rightStatus = String(right.dashboard_status ?? '').toLowerCase()
      const byStatus = (statusRank[leftStatus] ?? 99) - (statusRank[rightStatus] ?? 99)

      if (byStatus !== 0) {
        return byStatus
      }

      return getDateRank(left.execution_deadline) - getDateRank(right.execution_deadline)
    })
    .slice(0, 4)
})

const assistantActions = computed(() => {
  const actions = [
    {
      label: tr('Планы и отчёты', 'Жоспарлар мен есептер'),
      caption: tr('Открыть рабочий план', 'Жұмыс жоспарын ашу'),
      to: { name: 'plans' }
    },
    {
      label: tr('Профиль', 'Профиль'),
      caption: tr('Проверить учётные данные', 'Аккаунт деректерін тексеру'),
      to: { name: 'profile' }
    }
  ]

  if (isProrector.value || authStore.user?.role === 'admin') {
    actions.unshift({
      label: tr('Выполнение программы', 'Бағдарлама орындалуы'),
      caption: tr('Перейти к решениям по отчётам', 'Есептер бойынша шешімдерге өту'),
      to: { name: 'program-execution' }
    })
  }

  if (authStore.user?.role === 'admin') {
    actions.splice(1, 0, {
      label: tr('Плановый период', 'Жоспарлы кезең'),
      caption: tr('Обновить индикаторы по годам', 'Жылдар бойынша көрсеткіштерді жаңарту'),
      to: { name: 'planning-period' }
    })
  }

  return actions.slice(0, 4)
})

const listTitle = computed(() => {
  switch (activeCard.value) {
    case 'completed':
      return tr('Список принятых индикаторов', 'Қабылданған индикаторлар тізімі')
    case 'pending':
      return tr('Список индикаторов на проверке', 'Тексерістегі индикаторлар тізімі')
    case 'in_progress':
      return tr('Список индикаторов в работе', 'Жұмыстағы индикаторлар тізімі')
    case 'not_filled':
      return tr('Список незаполненных индикаторов', 'Толтырылмаған индикаторлар тізімі')
    case 'overdue':
      return tr('Список просроченных индикаторов', 'Мерзімі өткен индикаторлар тізімі')
    default:
      return tr('Список всех индикаторов', 'Барлық индикаторлар тізімі')
  }
})

function statusLabel(status) {
  const normalized = String(status ?? '').toLowerCase()
  if (normalized === 'completed') {
    return tr('Принято', 'Қабылданды')
  }
  if (normalized === 'pending') {
    return tr('На проверке', 'Тексерісте')
  }
  if (normalized === 'overdue') {
    return tr('Просрочено', 'Мерзімі өткен')
  }
  if (normalized === 'not_filled') {
    return tr('Не заполнено', 'Толтырылмаған')
  }
  return tr('В работе', 'Жұмыста')
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

function normalizeDateValue(value) {
  const raw = String(value ?? '').trim()
  if (!raw) {
    return ''
  }

  let normalized = raw
  normalized = normalized.replace(/^(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2}:\d{2})/, '$1T$2')
  normalized = normalized.replace(/([+-]\d{2})(\d{2})$/, '$1:$2')
  normalized = normalized.replace(/([+-]\d{2})$/, '$1:00')

  return normalized
}

function parseDateValue(value) {
  const normalized = normalizeDateValue(value)
  if (!normalized) {
    return null
  }

  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) {
    return null
  }

  return date
}

function getDateRank(value) {
  const date = parseDateValue(value)
  return date ? date.getTime() : Number.MAX_SAFE_INTEGER
}

function formatDate(value) {
  const raw = String(value ?? '').trim()
  if (!raw) {
    return '—'
  }

  const date = parseDateValue(value)
  if (!date) {
    return raw
  }

  return date.toLocaleString(tr('ru-RU', 'kk-KZ'))
}

function formatShortDate(value) {
  const raw = String(value ?? '').trim()
  if (!raw) {
    return '—'
  }

  const date = parseDateValue(value)
  if (!date) {
    return raw
  }

  return date.toLocaleDateString(tr('ru-RU', 'kk-KZ'), {
    day: 'numeric',
    month: 'short'
  })
}

function statusFilterByCard(cardKey) {
  switch (cardKey) {
    case 'completed':
      return 'completed'
    case 'pending':
      return 'pending'
    case 'in_progress':
      return 'in_progress'
    case 'not_filled':
      return 'not_filled'
    case 'overdue':
      return 'overdue'
    default:
      return ''
  }
}

function deriveDashboardStatus(row) {
  const reportStatus = String(row?.report_status ?? '').toLowerCase()
  if (reportStatus === 'pending') {
    return 'pending'
  }
  if (reportStatus === 'rejected') {
    return 'in_progress'
  }
  if (reportStatus === 'completed') {
    return 'completed'
  }

  const scheduleStatus = String(row?.schedule_status ?? '').toLowerCase()
  if (scheduleStatus === 'not_filled' || scheduleStatus === 'no_deadline') {
    return isProrector.value ? 'in_progress' : 'not_filled'
  }
  if (scheduleStatus === 'overdue') {
    return 'overdue'
  }

  return 'in_progress'
}

function recalculateStats() {
  const next = {
    total: allRows.value.length,
    completed: 0,
    pending: 0,
    not_filled: 0,
    in_progress: 0,
    overdue: 0
  }

  for (const row of allRows.value) {
    const status = String(row.dashboard_status ?? '').toLowerCase()
    if (status === 'completed') {
      next.completed += 1
      continue
    }
    if (status === 'pending') {
      next.pending += 1
      continue
    }
    if (status === 'overdue') {
      next.overdue += 1
      continue
    }
    if (status === 'not_filled') {
      next.not_filled += 1
      continue
    }
    next.in_progress += 1
  }

  stats.value = next
}

const rows = computed(() => {
  const filter = statusFilterByCard(activeCard.value)
  if (!filter) {
    return allRows.value
  }

  return allRows.value.filter((row) => String(row.dashboard_status).toLowerCase() === filter)
})

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
    return
  }

  const currentSelection = Number(selectedYear.value)
  if (Number.isInteger(currentSelection) && normalizedYears.includes(currentSelection)) {
    return
  }

  const preferredYear = normalizedYears.includes(currentYear)
    ? currentYear
    : normalizedYears[normalizedYears.length - 1]

  selectedYear.value = String(preferredYear)
}

async function loadRows() {
  if (!selectedYear.value) {
    allRows.value = []
    stats.value = {
      total: 0,
      completed: 0,
      pending: 0,
      not_filled: 0,
      in_progress: 0,
      overdue: 0
    }
    return
  }

  const response = await fetchPlanIndicators(selectedYear.value, {
    include_submitted: isProrector.value
  })

  allRows.value = (response.items ?? []).map((item) => ({
    ...item,
    dashboard_status: deriveDashboardStatus(item)
  }))
  recalculateStats()
}

async function initialize() {
  loading.value = true
  errorMessage.value = ''

  try {
    await loadYears()
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Ошибка загрузки панели управления', 'Басқару панелін жүктеу кезінде қате болды')
  } finally {
    loading.value = false
  }
}

async function handleYearChange(event) {
  selectedYear.value = event.target.value
  loading.value = true
  errorMessage.value = ''

  try {
    await loadRows()
  } catch (error) {
    errorMessage.value = error?.response?.data?.error
      ?? (typeof error?.response?.data === 'string' ? error.response.data : null)
      ?? error?.message
      ?? tr('Не удалось загрузить данные по году', 'Жыл бойынша деректі жүктеу мүмкін болмады')
  } finally {
    loading.value = false
  }
}

function handleCardClick(cardKey) {
  if (activeCard.value === cardKey) {
    return
  }

  activeCard.value = cardKey
}

onMounted(() => {
  initialize()
})
</script>

<template>
  <section class="dashboard-page">
    <PageHeader
      :title="tr('Панель управления', 'Басқару панелі')"
      :subtitle="tr('Ключевой обзор по индикаторам, срокам исполнения и последним отправленным отчетам за активный год', 'Белсенді жыл бойынша индикаторлар, мерзімдер және соңғы жіберілген есептерге шолу')"
      :eyebrow="tr('Обзор', 'Шолу')"
    />

    <div class="dashboard-overview-layout">
      <div class="dashboard-stats">
        <button
          v-for="card in cards"
          :key="card.key"
          type="button"
          class="unstyled-button dashboard-stat-card"
          :class="{ 'is-active': activeCard === card.key }"
          @click="handleCardClick(card.key)"
        >
          <div class="dashboard-stat-head">
            <span class="dashboard-stat-label">{{ card.label }}</span>
            <span class="dashboard-stat-chip">{{ selectedYear || '—' }}</span>
          </div>
          <strong>{{ card.value }}</strong>
          <p>{{ card.meta }}</p>
        </button>
      </div>

      <section class="panel panel-strong dashboard-filter-card">
        <div class="dashboard-filter-card-head">
          <div>
            <span class="kicker">{{ tr('Активный год', 'Белсенді жыл') }}</span>
            <h3>{{ selectedYear || tr('Нет года', 'Жыл жоқ') }}</h3>
          </div>
          <strong>{{ completionRateText }}</strong>
        </div>

        <label class="dashboard-filter">
          <span>{{ tr('Год', 'Жыл') }}</span>
          <select :value="selectedYear" :disabled="loading || !hasYears" @change="handleYearChange">
            <option v-if="!hasYears" value="">{{ tr('Нет годов', 'Жылдар жоқ') }}</option>
            <option v-for="year in years" :key="year" :value="String(year)">
              {{ year }}
            </option>
          </select>
        </label>

        <p class="dashboard-filter-note">
          {{ activeCardDetails?.meta || tr('Выберите карточку статуса, чтобы мгновенно менять фокус списка и правого приоритетного блока.', 'Тізім мен оң жақ приоритет блогының фокусын бірден өзгерту үшін мәртебе карточкасын таңдаңыз.') }}
        </p>

        <div class="dashboard-filter-progress">
          <div>
            <span>{{ tr('Принято', 'Қабылданды') }}</span>
            <strong>{{ stats.completed }}/{{ stats.total }}</strong>
          </div>
          <div>
            <span>{{ tr('Просрочено', 'Мерзімі өткен') }}</span>
            <strong>{{ stats.overdue }}</strong>
          </div>
        </div>
      </section>
    </div>

    <p v-if="errorMessage" class="message message-error">{{ errorMessage }}</p>

    <div class="dashboard-content-grid">
      <section class="panel panel-strong dashboard-analytics-card">
        <div class="panel-header dashboard-analytics-head">
          <div>
            <h3 class="panel-title">{{ tr('Пульс исполнения', 'Орындау пульсі') }}</h3>
            <p class="panel-subtitle">{{ tr('Срез по активному году показывает, где подтверждение идёт стабильно, а где нужен ручной контроль.', 'Белсенді жыл кескіні қай жерде растау тұрақты, ал қай жерде қолмен бақылау қажет екенін көрсетеді.') }}</p>
          </div>

          <div class="dashboard-analytics-summary">
            <span class="kicker">{{ activeCardDetails?.label || tr('Фокус', 'Фокус') }}</span>
            <strong>{{ completionRateText }}</strong>
            <span>{{ tr('доля принятых задач', 'қабылданған тапсырмалар үлесі') }}</span>
          </div>
        </div>

        <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
        <div v-else-if="!hasYears" class="empty-state">{{ tr('Пока нет данных по годам', 'Әзірге жылдық дерек жоқ') }}</div>
        <div v-else class="dashboard-analytics-body">
          <div class="dashboard-analytics-copy">
            <article class="dashboard-performance-card">
              <span class="dashboard-performance-label">{{ tr('Текущий фокус', 'Ағымдағы фокус') }}</span>
              <strong>{{ activeCardDetails?.label || tr('Все задачи', 'Барлық тапсырма') }}</strong>
              <p>{{ activeCardDetails?.meta || tr('Список ниже автоматически перестраивается по выбранной карточке статуса.', 'Төмендегі тізім таңдалған мәртебе карточкасына сай автоматты қайта құрылады.') }}</p>
            </article>

            <div class="dashboard-insights-list">
              <article
                v-for="item in dashboardSeries"
                :key="item.key"
                class="dashboard-insight-item"
              >
                <span class="dashboard-insight-swatch" :class="item.tone"></span>
                <div class="dashboard-insight-copy">
                  <strong>{{ item.value }}</strong>
                  <span>{{ item.label }}</span>
                </div>
                <small>{{ item.caption }}</small>
              </article>
            </div>
          </div>

          <div class="dashboard-chart-shell">
            <div class="dashboard-chart-grid">
              <article
                v-for="item in dashboardSeries"
                :key="`chart-${item.key}`"
                class="dashboard-chart-column"
              >
                <div class="dashboard-chart-dots">
                  <span
                    v-for="dotIndex in item.dotCount"
                    :key="`${item.key}-${dotIndex}`"
                    class="dashboard-chart-dot"
                    :class="item.tone"
                  ></span>
                </div>
                <strong>{{ item.value }}</strong>
                <span>{{ item.label }}</span>
              </article>
            </div>
          </div>
        </div>
      </section>

      <aside class="dashboard-side-column">
        <section class="panel panel-strong dashboard-priority-card">
          <div class="panel-header">
            <div>
              <h3 class="panel-title">{{ tr('Приоритетные задачи', 'Приоритетті тапсырмалар') }}</h3>
              <p class="panel-subtitle">{{ tr('Сначала показаны просроченные и ожидающие проверки позиции.', 'Алдымен мерзімі өткен және тексеруді күтіп тұрған позициялар көрсетіледі.') }}</p>
            </div>
            <span class="kicker">{{ priorityRows.length }}</span>
          </div>

          <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
          <div v-else-if="priorityRows.length === 0" class="empty-state">{{ tr('Нет активных приоритетов', 'Белсенді приоритеттер жоқ') }}</div>
          <div v-else class="dashboard-priority-list">
            <article
              v-for="row in priorityRows"
              :key="`priority-${row.indicator_id}`"
              class="dashboard-priority-item"
            >
              <span class="dashboard-priority-marker" :class="`is-${row.dashboard_status}`"></span>

              <div class="dashboard-priority-body">
                <strong>{{ textPreview(row.development_indicator) }}</strong>
                <div class="dashboard-priority-meta">
                  <span>{{ statusLabel(row.dashboard_status) }}</span>
                  <span>{{ formatShortDate(row.execution_deadline) }}</span>
                </div>
                <p>{{ row.responsible || tr('Ответственный не указан', 'Жауапты көрсетілмеген') }}</p>
              </div>
            </article>
          </div>
        </section>

        <section class="panel panel-accent dashboard-assistant-card">
          <span class="dashboard-assistant-eyebrow">{{ tr('Привет,', 'Сәлем,') }} {{ currentUserFirstName }}</span>
          <h3>{{ tr('Что открыть дальше?', 'Келесі не ашамыз?') }}</h3>
          <p>{{ tr('Быстрые переходы оставляют рабочий сценарий рядом с обзором, чтобы не терять контекст.', 'Жылдам өтулер жұмыс сценарийін шолуға жақын ұстап, контексті жоғалтпауға көмектеседі.') }}</p>

          <div class="dashboard-assistant-links">
            <RouterLink
              v-for="action in assistantActions"
              :key="action.label"
              :to="action.to"
              class="dashboard-assistant-link"
            >
              <strong>{{ action.label }}</strong>
              <span>{{ action.caption }}</span>
            </RouterLink>
          </div>
        </section>
      </aside>
    </div>

    <section class="panel panel-strong dashboard-list-card">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ listTitle }}</h3>
          <p class="panel-subtitle">{{ tr('Список автоматически перестраивается по выбранной карточке статуса.', 'Тізім таңдалған мәртебе карточкасына сай автоматты қайта құрылады.') }}</p>
        </div>
        <span class="kicker">{{ rows.length }} {{ tr('строк', 'жол') }}</span>
      </div>

      <div v-if="loading" class="empty-state">{{ tr('Загрузка...', 'Жүктелуде...') }}</div>
      <div v-else-if="!hasYears" class="empty-state">{{ tr('Пока нет данных по годам', 'Әзірге жылдық дерек жоқ') }}</div>
      <div v-else-if="rows.length === 0" class="empty-state">{{ tr('Список пуст', 'Тізім бос') }}</div>
      <div v-else class="table-wrap">
        <table class="table dashboard-table">
          <thead>
            <tr>
              <th>№</th>
              <th>{{ tr('Целевой индикатор', 'Мақсатты индикатор') }}</th>
              <th>{{ tr('Срок исполнения', 'Орындау мерзімі') }}</th>
              <th>{{ tr('Ответственные', 'Жауаптылар') }}</th>
              <th>{{ tr('Статус', 'Мәртебе') }}</th>
              <th>{{ tr('Отправлено', 'Жіберілген уақыты') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in rows" :key="row.indicator_id">
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
              <td :data-label="tr('Статус', 'Мәртебе')">
                <span class="status-pill" :class="`status-${row.dashboard_status}`">
                  {{ statusLabel(row.dashboard_status) }}
                </span>
              </td>
              <td :data-label="tr('Отправлено', 'Жіберілген уақыты')">{{ formatDate(row.last_submitted_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div v-if="readModalOpen" class="modal-backdrop" @click.self="closeReadModal">
      <div class="modal-card dashboard-read-modal">
        <h3 class="modal-title">{{ readModalTitle }}</h3>
        <div class="dashboard-read-content text-pretty">
          {{ readModalText }}
        </div>
        <div class="modal-actions">
          <button class="btn btn-primary" type="button" @click="closeReadModal">{{ tr('Закрыть', 'Жабу') }}</button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.dashboard-overview-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 310px;
  gap: 1rem;
}

.dashboard-stats {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(auto-fit, minmax(188px, 1fr));
}

.dashboard-stat-card {
  display: grid;
  gap: 0.7rem;
  padding: 1.2rem;
  border-radius: 26px;
  border: 1px solid rgba(222, 225, 242, 0.98);
  background:
    radial-gradient(circle at top right, rgba(160, 235, 225, 0.26), transparent 34%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(247, 248, 255, 0.94));
  box-shadow: 0 16px 38px rgba(104, 114, 170, 0.1);
  text-align: left;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.dashboard-stat-card:hover {
  transform: translateY(-2px);
}

.dashboard-stat-card.is-active {
  border-color: rgba(145, 140, 245, 0.5);
  box-shadow: 0 22px 44px rgba(114, 118, 205, 0.16);
}

.dashboard-stat-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.dashboard-stat-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 1.8rem;
  padding: 0.2rem 0.62rem;
  border-radius: 999px;
  background: rgba(240, 242, 255, 0.96);
  color: var(--muted-strong);
  font-size: 0.72rem;
  font-weight: 700;
}

.dashboard-stat-label {
  color: var(--muted);
  font-size: 0.79rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.dashboard-stat-card strong {
  font-size: clamp(2.05rem, 4vw, 2.9rem);
  line-height: 0.92;
  letter-spacing: -0.05em;
}

.dashboard-stat-card p {
  margin: 0;
  color: var(--muted);
  font-size: 0.88rem;
}

.dashboard-filter-card {
  display: grid;
  gap: 1rem;
  padding: 1.2rem;
}

.dashboard-filter-card-head {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
}

.dashboard-filter-card-head h3 {
  margin: 0.45rem 0 0;
  font-size: 1.7rem;
  line-height: 1;
  letter-spacing: -0.05em;
}

.dashboard-filter-card-head strong {
  font-size: 1.9rem;
  line-height: 1;
  letter-spacing: -0.05em;
}

.dashboard-filter {
  min-width: 0;
}

.dashboard-filter-note {
  margin: 0;
  color: var(--muted);
  font-size: 0.9rem;
}

.dashboard-filter-progress {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.dashboard-filter-progress div {
  display: grid;
  gap: 0.18rem;
  padding: 0.85rem 0.9rem;
  border-radius: 20px;
  background: rgba(247, 248, 255, 0.96);
  border: 1px solid rgba(225, 228, 244, 0.96);
}

.dashboard-filter-progress span {
  color: var(--muted);
  font-size: 0.76rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.dashboard-filter-progress strong {
  font-size: 1.15rem;
}

.dashboard-content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) 330px;
  gap: 1rem;
  align-items: start;
}

.dashboard-analytics-card,
.dashboard-priority-card,
.dashboard-assistant-card,
.dashboard-list-card {
  padding: 1.2rem;
}

.dashboard-analytics-head {
  align-items: flex-start;
}

.dashboard-analytics-summary {
  display: grid;
  gap: 0.25rem;
  justify-items: end;
  text-align: right;
}

.dashboard-analytics-summary strong {
  font-size: 2rem;
  line-height: 1;
  letter-spacing: -0.05em;
}

.dashboard-analytics-summary span:last-child {
  color: var(--muted);
  font-size: 0.82rem;
}

.dashboard-analytics-body {
  display: grid;
  grid-template-columns: minmax(250px, 0.78fr) minmax(0, 1fr);
  gap: 1rem;
  align-items: stretch;
}

.dashboard-analytics-copy {
  display: grid;
  gap: 0.95rem;
}

.dashboard-performance-card {
  display: grid;
  gap: 0.55rem;
  padding: 1rem;
  border-radius: 24px;
  background:
    radial-gradient(circle at top right, rgba(160, 235, 225, 0.28), transparent 34%),
    linear-gradient(180deg, rgba(246, 248, 255, 0.98), rgba(255, 255, 255, 0.96));
  border: 1px solid rgba(221, 224, 243, 0.98);
}

.dashboard-performance-label {
  color: var(--muted);
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.dashboard-performance-card strong {
  font-size: 1.55rem;
  line-height: 1;
  letter-spacing: -0.04em;
}

.dashboard-performance-card p {
  margin: 0;
  color: var(--muted);
  font-size: 0.92rem;
}

.dashboard-insights-list {
  display: grid;
  gap: 0.75rem;
}

.dashboard-insight-item {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 0.75rem;
  align-items: center;
  padding: 0.82rem 0.9rem;
  border-radius: 22px;
  background: rgba(248, 249, 255, 0.88);
  border: 1px solid rgba(225, 228, 244, 0.96);
}

.dashboard-insight-swatch {
  width: 0.9rem;
  height: 0.9rem;
  border-radius: 50%;
}

.dashboard-insight-swatch.is-success,
.dashboard-chart-dot.is-success {
  background: #69d8c0;
}

.dashboard-insight-swatch.is-accent,
.dashboard-chart-dot.is-accent {
  background: #8c83ff;
}

.dashboard-insight-swatch.is-warning,
.dashboard-chart-dot.is-warning {
  background: #f0bb71;
}

.dashboard-insight-swatch.is-danger,
.dashboard-chart-dot.is-danger {
  background: #f08f97;
}

.dashboard-insight-swatch.is-muted,
.dashboard-chart-dot.is-muted {
  background: #c7cde5;
}

.dashboard-insight-copy {
  display: grid;
  gap: 0.06rem;
}

.dashboard-insight-copy strong {
  font-size: 1.02rem;
}

.dashboard-insight-copy span,
.dashboard-insight-item small {
  color: var(--muted);
}

.dashboard-insight-item small {
  text-align: right;
}

.dashboard-chart-shell {
  min-width: 0;
  padding: 1rem;
  border-radius: 28px;
  border: 1px solid rgba(223, 226, 244, 0.98);
  background:
    radial-gradient(circle at top left, rgba(125, 121, 243, 0.08), transparent 26%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(245, 247, 255, 0.96));
}

.dashboard-chart-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 0.8rem;
  align-items: end;
  min-height: 100%;
}

.dashboard-chart-column {
  display: grid;
  gap: 0.65rem;
  justify-items: center;
  text-align: center;
}

.dashboard-chart-column strong {
  font-size: 1.2rem;
  line-height: 1;
}

.dashboard-chart-column span {
  color: var(--muted);
  font-size: 0.8rem;
}

.dashboard-chart-dots {
  display: grid;
  grid-template-columns: repeat(2, 14px);
  gap: 0.45rem;
  align-content: end;
  justify-content: center;
  min-height: 220px;
  width: 100%;
  padding: 1rem 0 0.25rem;
}

.dashboard-chart-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  box-shadow: 0 6px 16px rgba(102, 108, 166, 0.12);
}

.dashboard-side-column {
  display: grid;
  gap: 1rem;
}

.dashboard-priority-list {
  display: grid;
  gap: 0.75rem;
}

.dashboard-priority-item {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.8rem;
  align-items: flex-start;
  padding: 0.95rem;
  border-radius: 22px;
  border: 1px solid rgba(224, 227, 244, 0.96);
  background: rgba(248, 249, 255, 0.88);
}

.dashboard-priority-marker {
  width: 0.82rem;
  height: 0.82rem;
  margin-top: 0.38rem;
  border-radius: 50%;
  background: #c7cde5;
}

.dashboard-priority-marker.is-completed {
  background: #69d8c0;
}

.dashboard-priority-marker.is-pending {
  background: #8c83ff;
}

.dashboard-priority-marker.is-in_progress {
  background: #f0bb71;
}

.dashboard-priority-marker.is-overdue {
  background: #f08f97;
}

.dashboard-priority-marker.is-not_filled {
  background: #c7cde5;
}

.dashboard-priority-body {
  display: grid;
  gap: 0.45rem;
}

.dashboard-priority-body strong {
  line-height: 1.4;
}

.dashboard-priority-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  color: var(--muted);
  font-size: 0.8rem;
}

.dashboard-priority-body p {
  margin: 0;
  color: var(--muted-strong);
  font-size: 0.88rem;
}

.dashboard-assistant-card {
  display: grid;
  gap: 1rem;
}

.dashboard-assistant-eyebrow {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 0.42rem 0.72rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.82);
  color: var(--accent-strong);
  font-size: 0.78rem;
  font-weight: 700;
}

.dashboard-assistant-card h3 {
  margin: 0;
  font-size: 2rem;
  line-height: 1;
  letter-spacing: -0.05em;
}

.dashboard-assistant-card p {
  margin: 0;
  color: var(--muted-strong);
}

.dashboard-assistant-links {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.dashboard-assistant-link {
  display: grid;
  gap: 0.25rem;
  padding: 0.95rem;
  border-radius: 22px;
  border: 1px solid rgba(222, 225, 243, 0.96);
  background: rgba(255, 255, 255, 0.88);
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.dashboard-assistant-link strong {
  font-size: 0.94rem;
}

.dashboard-assistant-link span {
  color: var(--muted);
  font-size: 0.82rem;
}

.dashboard-assistant-link:hover {
  transform: translateY(-1px);
  box-shadow: 0 16px 34px rgba(106, 117, 180, 0.12);
}

.dashboard-table {
  min-width: 900px;
}

.number-cell {
  width: 4rem;
  text-align: center;
  font-weight: 700;
}

.planned-value-chip {
  display: inline-flex;
  align-items: center;
  margin-top: 0.4rem;
  border: 1px solid rgba(220, 223, 242, 0.98);
  border-radius: 999px;
  padding: 0.24rem 0.62rem;
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--muted-strong);
  background: rgba(248, 249, 255, 0.94);
}

.dashboard-read-modal {
  width: min(760px, 100%);
}

.dashboard-read-content {
  max-height: min(60vh, 460px);
  overflow: auto;
  margin-top: 0.6rem;
  padding: 0.9rem 1rem;
  border-radius: 14px;
  border: 1px solid rgba(220, 223, 242, 0.98);
  background: rgba(248, 249, 255, 0.88);
  white-space: pre-wrap;
  line-height: 1.5;
}

@media (max-width: 1160px) {
  .dashboard-overview-layout,
  .dashboard-content-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 840px) {
  .dashboard-filter-progress,
  .dashboard-assistant-links,
  .dashboard-analytics-body {
    grid-template-columns: 1fr;
  }

  .dashboard-list-card {
    padding: 0.95rem;
  }

  .dashboard-chart-grid {
    grid-template-columns: repeat(auto-fit, minmax(92px, 1fr));
  }

  .dashboard-chart-dots {
    min-height: 120px;
  }

  .table-wrap {
    overflow: visible;
    border: 0;
    box-shadow: none;
    background: transparent;
  }

  .dashboard-table {
    min-width: 0;
    display: block;
  }

  .dashboard-table thead {
    display: none;
  }

  .dashboard-table tbody {
    display: grid;
    gap: 0.78rem;
  }

  .dashboard-table tbody tr {
    display: block;
    padding: 0.72rem 0.82rem;
    border: 1px solid var(--border);
    border-radius: 18px;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: var(--shadow-soft);
  }

  .dashboard-table tbody td {
    display: grid;
    grid-template-columns: minmax(120px, 38%) 1fr;
    gap: 0.52rem;
    padding: 0.48rem 0.1rem;
    border-bottom: 1px dashed rgba(215, 220, 241, 0.96);
  }

  .dashboard-table tbody td:last-child {
    border-bottom: 0;
  }

  .dashboard-table tbody td::before {
    content: attr(data-label);
    color: var(--muted);
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .number-cell {
    text-align: left;
  }
}

@media (max-width: 620px) {
  .dashboard-stats,
  .dashboard-assistant-links {
    grid-template-columns: 1fr;
  }

  .dashboard-table tbody td {
    grid-template-columns: 1fr;
    gap: 0.38rem;
  }

  .dashboard-table tbody td::before {
    font-size: 0.68rem;
  }
}
</style>
