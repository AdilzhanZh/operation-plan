import http from './http'

export async function fetchTasks(params = {}) {
  const { data } = await http.get('/tasks', { params })
  return data
}

export async function createTask(payload) {
  const { data } = await http.post('/tasks', payload)
  return data
}

export async function updateTask(id, payload) {
  const { data } = await http.patch(`/tasks/${id}`, payload)
  return data
}

export async function updateTaskStatus(id, status, comment = '') {
  const { data } = await http.patch(`/tasks/${id}/status`, { status, comment })
  return data
}
