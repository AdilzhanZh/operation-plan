import http from './http'

export async function loginRequest(payload) {
  const { data } = await http.post('/login', payload)
  return data
}

export async function requestRegistrationCode(payload) {
  const { data } = await http.post('/register/request-code', payload)
  return data
}

export async function verifyRegistrationCode(payload) {
  const { data } = await http.post('/register/verify-code', payload)
  return data
}

export async function registerRequest(payload) {
  return requestRegistrationCode(payload)
}

export async function requestPasswordResetCode(payload) {
  const { data } = await http.post('/password-reset/request-code', payload)
  return data
}

export async function verifyPasswordResetCode(payload) {
  const { data } = await http.post('/password-reset/verify-code', payload)
  return data
}

export async function completePasswordResetRequest(payload) {
  const { data } = await http.post('/password-reset/confirm', payload)
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
