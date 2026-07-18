import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  withCredentials: true,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => {
    if (response.data?.data?.access_token) {
      localStorage.setItem('access_token', response.data.data.access_token)
      if (response.data.data.refresh_token) {
        localStorage.setItem('refresh_token', response.data.data.refresh_token)
      }
    }
    return response
  },
  async (error) => {
    if (error.response?.status === 401) {
      const requestUrl = error.config?.url || ''
      const isAuthRequest = [
        '/user/login',
        '/user/register',
        '/user/forgot-password',
        '/user/reset-password',
      ].some((path) => requestUrl.includes(path))

      if (isAuthRequest) {
        return Promise.reject(error)
      }

      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken && !error.config._retry) {
        error.config._retry = true
        try {
          const res = await axios.post('/api/v1/user/refresh', { refresh_token: refreshToken }, {
            withCredentials: true,
          })
          const { access_token, refresh_token } = res.data.data
          localStorage.setItem('access_token', access_token)
          localStorage.setItem('refresh_token', refresh_token)
          error.config.headers.Authorization = `Bearer ${access_token}`
          return api(error.config)
        } catch {
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          window.location.href = '/login'
        }
      } else if (!error.config._retry) {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  },
)

export default api
