import http from './http'

export async function fetchPlanYears() {
  const { data } = await http.get('/plans/years')
  return data
}

export async function fetchPlanIndicators(year) {
  const { data } = await http.get('/plans/indicators', {
    params: { year }
  })
  return data
}

export async function savePlanIndicator(indicatorId, year, payload) {
  const { data } = await http.put(`/plans/indicators/${indicatorId}`, payload, {
    params: { year }
  })
  return data
}
