import http from './http'

export async function fetchUsersRequest() {
  const { data } = await http.get('/users')
  return data
}

export async function fetchProrectorsRequest() {
  const { data } = await http.get('/users/prorectors')
  return data
}

export async function createUserRequest(payload) {
  const { data } = await http.post('/users', payload)
  return data
}
