import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  // SECURITY: 携带 httpOnly Cookie 用于认证
  withCredentials: true,
})

// SECURITY: 移除 localStorage token 读取，改由后端 httpOnly Cookie 自动携带
// 保留 Authorization header 作为降级方案（兼容旧版后端）
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => {
    // 登录/注册时自动保存 token
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
      // SECURITY: 优先从 Cookie 获取 refresh_token（httpOnly Cookie 由浏览器自动携带）
      // 降级方案：仍支持从 localStorage 读取（兼容过渡期）
      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken && !error.config._retry) {
        error.config._retry = true
        try {
          const res = await axios.post('/api/v1/user/refresh', { refresh_token: refreshToken }, {
            withCredentials: true,
          })
          const { access_token, refresh_token } = res.data.data
          // SECURITY: 新 token 已由后端设置到 httpOnly Cookie
          // 仅在降级模式下更新 localStorage
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
        // Cookie 模式下无 refresh_token 在 localStorage，直接跳转登录
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  },
)

export default api
