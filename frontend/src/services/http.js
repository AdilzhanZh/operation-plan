import axios from 'axios'
import { useAuthStore } from '../store/auth'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '/api',
  timeout: 20000
})

export async function fetchAllPaginated(path, params = {}) {
  const items = []
  let page = 1
  let totalPages = 1
  let lastData = null

  do {
    const { data } = await http.get(path, {
      params: {
        ...params,
        page,
        limit: 100
      }
    })

    lastData = data
    if (Array.isArray(data?.items)) {
      items.push(...data.items)
    }

    const nextTotalPages = Number(data?.meta?.total_pages)
    if (!Number.isInteger(nextTotalPages) || nextTotalPages < 1) {
      break
    }

    totalPages = nextTotalPages
    page += 1
  } while (page <= totalPages)

  if (!lastData || !Array.isArray(lastData.items)) {
    return lastData
  }

  return {
    ...lastData,
    items,
    meta: {
      ...(lastData.meta ?? {}),
      page: items.length > 0 ? 1 : 0,
      limit: items.length,
      total: items.length,
      total_pages: items.length > 0 ? 1 : 0
    }
  }
}

http.interceptors.request.use((config) => {
  const authStore = useAuthStore()

  if (authStore.token) {
    config.headers.Authorization = `Bearer ${authStore.token}`
  }

  return config
})

export default http
