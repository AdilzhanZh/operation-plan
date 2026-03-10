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

export async function submitPlanIndicatorReport(indicatorId, year, payload) {
  const formData = new FormData()
  if (payload?.report_text) {
    formData.append('report_text', payload.report_text)
  }
  if (payload?.file) {
    formData.append('file', payload.file)
  }

  const { data } = await http.post(`/plans/indicators/${indicatorId}/report`, formData, {
    params: { year },
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
  return data
}

export async function fetchPlanReports(year) {
  const { data } = await http.get('/plans/reports', {
    params: { year }
  })
  return data
}
