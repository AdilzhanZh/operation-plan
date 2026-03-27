import http, { fetchAllPaginated } from './http'

export async function fetchUsersRequest(params = {}) {
  if (params?.page || params?.limit) {
    const { data } = await http.get('/users', { params })
    return data
  }

  return fetchAllPaginated('/users', params)
}

export async function fetchProrectorsRequest() {
  const { data } = await http.get('/users/prorectors')
  return data
}

export async function createUserRequest(payload) {
  const { data } = await http.post('/users', payload)
  return data
}

export async function deleteUserRequest(id) {
  const { data } = await http.delete(`/users/${id}`)
  return data
}

export async function fetchRegistrationRequestsRequest(params = {}) {
  if (params?.page || params?.limit) {
    const { data } = await http.get('/registration-requests', { params })
    return data
  }

  return fetchAllPaginated('/registration-requests', params)
}

export async function approveRegistrationRequest(id) {
  const { data } = await http.patch(`/registration-requests/${id}/approve`)
  return data
}

export async function rejectRegistrationRequest(id, payload = {}) {
  const { data } = await http.patch(`/registration-requests/${id}/reject`, payload)
  return data
}
