import http from './http'

export async function fetchPlanningPeriod() {
  const { data } = await http.get('/planning-period')
  return data
}

export async function createPlanningPeriodIndicator(payload) {
  const { data } = await http.post('/planning-period', payload)
  return data
}

export async function updatePlanningPeriodIndicator(id, payload) {
  const { data } = await http.patch(`/planning-period/${id}`, payload)
  return data
}

export async function importPlanningPeriodExcel(file) {
  const formData = new FormData()
  formData.append('file', file)

  const { data } = await http.post('/planning-period/import', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
  return data
}
