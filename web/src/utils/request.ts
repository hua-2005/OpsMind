/**
 * Axios 实例封装
 *
 * 创建统一的 HTTP 客户端，配置：
 * - 请求拦截器：注入 Authorization: Bearer <token>
 * - 响应拦截器：处理 401（跳转登录）、403（提示无权限）、统一提取 data
 */

import axios from 'axios'
import { getToken, removeToken } from './auth'
import router from '@/router'

// 创建 Axios 实例，baseURL 为空（通过 Vite proxy 转发）
const request = axios.create({
  timeout: 30000
})

// 请求拦截器：注入 token
request.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器：统一错误处理
request.interceptors.response.use(
  (response) => {
    // 统一提取 data 字段
    return response.data
  },
  (error) => {
    const { response } = error

    if (response) {
      switch (response.status) {
        case 401:
          // 未登录或令牌过期，清除 token 并跳转登录页
          removeToken()
          router.push('/login')
          break
        case 403:
          // 无权限
          console.error('无权限访问')
          break
        default:
          console.error(response.data?.message || '请求失败')
      }
    } else {
      console.error('网络错误')
    }

    return Promise.reject(error)
  }
)

export default request
