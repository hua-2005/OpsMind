// Pinia 认证状态：token、用户信息、权限、登录/退出逻辑
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, refresh, logout as logoutApi, getProfile, getPermissions } from '@/api/auth'
import type { LoginResult, MenuItem } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>('')
  const user = ref<LoginResult['user'] | null>(null)
  const roles = ref<string[]>([])
  const permissions = ref<string[]>([])
  const menus = ref<MenuItem[]>([])

  const isLoggedIn = computed(() => !!token.value)

  // 从 localStorage 恢复登录态
  function restoreFromStorage() {
    const saved = localStorage.getItem('opsmind_auth')
    if (saved) {
      try {
        const data = JSON.parse(saved)
        token.value = data.token || ''
        user.value = data.user || null
        roles.value = data.roles || []
        permissions.value = data.permissions || []
        menus.value = data.menus || []
      } catch {
        clearAuth()
      }
    }
  }

  function saveToStorage() {
    localStorage.setItem('opsmind_auth', JSON.stringify({
      token: token.value,
      user: user.value,
      roles: roles.value,
      permissions: permissions.value,
      menus: menus.value,
    }))
  }

  function clearAuth() {
    token.value = ''
    user.value = null
    roles.value = []
    permissions.value = []
    menus.value = []
    localStorage.removeItem('opsmind_auth')
  }

  async function login(username: string, password: string) {
    const res = await loginApi({ username, password })
    const d = res.data.data
    token.value = d.access_token
    user.value = d.user
    roles.value = d.roles
    permissions.value = d.permissions
    menus.value = [] // 登录后单独加载菜单树
    saveToStorage()
    return d
  }

  // 登录后加载完整权限（含菜单树）
  async function loadPermissions() {
    try {
      const res = await getPermissions()
      menus.value = res.data.data.menus
      permissions.value = [
        ...new Set([...permissions.value, ...res.data.data.buttons]),
      ]
      saveToStorage()
    } catch {
      // 权限加载失败不阻断登录流程
    }
  }

  async function logout() {
    try {
      await logoutApi()
    } catch {
      // 退出请求失败也清除本地状态
    }
    clearAuth()
  }

  // 检查是否拥有指定权限
  function hasPermission(code: string): boolean {
    return permissions.value.includes(code)
  }

  return {
    token, user, roles, permissions, menus, isLoggedIn,
    restoreFromStorage, login, loadPermissions, logout, hasPermission, clearAuth,
  }
})
