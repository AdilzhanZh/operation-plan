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
