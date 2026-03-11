import http from './http'

export async function loginRequest(payload) {
  const { data } = await http.post('/login', payload)
  return data
}

export async function registerRequest(payload) {
  const { data } = await http.post('/register', payload)
  return data
}

export async function getMeRequest() {
  const { data } = await http.get('/me')
  return data
}

export async function changePasswordRequest(payload) {
  const { data } = await http.post('/change-password', payload)
  return data
}

export async function logoutRequest() {
  const { data } = await http.post('/logout')
  return data
}
