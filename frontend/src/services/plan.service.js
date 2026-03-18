import http, { fetchAllPaginated } from './http'

export async function fetchPlanYears() {
  const { data } = await http.get('/plans/years')
  return data
}

export async function fetchPlanIndicators(year, options = {}) {
  const params = { year }
  if (options?.include_submitted !== undefined) {
    params.include_submitted = options.include_submitted
  }
  if (options?.q) {
    params.q = options.q
  }
  if (options?.direction) {
    params.direction = options.direction
  }
  if (options?.page) {
    params.page = options.page
  }
  if (options?.limit) {
    params.limit = options.limit
  }

  if (options?.page || options?.limit) {
    const { data } = await http.get('/plans/indicators', { params })
    return data
  }

  return fetchAllPaginated('/plans/indicators', params)
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

  if (Array.isArray(payload?.files)) {
    for (const file of payload.files) {
      if (file) {
        formData.append('files', file)
      }
    }
  } else if (payload?.file) {
    formData.append('files', payload.file)
  }

  const { data } = await http.post(`/plans/indicators/${indicatorId}/report`, formData, {
    params: { year },
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
  return data
}

export async function fetchPlanReports(year, options = {}) {
  const params = { year }
  if (options?.status) {
    params.status = options.status
  }
  if (options?.q) {
    params.q = options.q
  }
  if (options?.page) {
    params.page = options.page
  }
  if (options?.limit) {
    params.limit = options.limit
  }
  if (options?.submitted_by) {
    params.submitted_by = options.submitted_by
  }
  if (options?.indicator_id) {
    params.indicator_id = options.indicator_id
  }

  if (options?.page || options?.limit) {
    const { data } = await http.get('/plans/reports', { params })
    return data
  }

  return fetchAllPaginated('/plans/reports', params)
}

export async function reviewPlanReport(reportId, payload) {
  const { data } = await http.patch(`/plans/reports/${reportId}/review`, payload)
  return data
}

export async function updatePlanReport(reportId, payload) {
  const { data } = await http.patch(`/plans/reports/${reportId}`, payload)
  return data
}

export async function downloadPlanReportFile(fileId) {
  const response = await http.get(`/plans/reports/files/${fileId}/download`, {
    responseType: 'blob'
  })
  return response
}
