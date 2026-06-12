import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getToken, setToken as saveToken, removeToken } from '../utils/auth'
import type { MenuItem } from '@/types/menu'

interface UserInfo {
  id: number
  username: string
  real_name: string
  phone: string
  email: string
  status?: number    // 登录接口不返回，用户管理接口返回
  first_login: boolean
}

export type { MenuItem }

export const useAuthStore = defineStore('auth', () => {
  // TODO(store/auth): 刷新页面后只恢复 token，不恢复 user/roles/permissions/menus。
  // 路由守卫和布局菜单会在首次刷新时丢失权限状态。
  // State
  const token = ref(getToken() || '')
  const user = ref<UserInfo | null>(null)
  const roles = ref<string[]>([])
  const permissions = ref<string[]>([])
  const menus = ref<MenuItem[]>([])

  // Getters
  const isLoggedIn = computed(() => !!token.value)

  const hasPermission = (perm: string) => {
    return permissions.value.includes(perm)
  }

  // Actions
  const setToken = (newToken: string) => {
    token.value = newToken
    saveToken(newToken)
  }

  const clearAuth = () => {
    token.value = ''
    user.value = null
    roles.value = []
    permissions.value = []
    menus.value = []
    removeToken()
  }

  const setUserInfo = (data: {
    user: UserInfo
    roles: string[]
    permissions: string[]
    menus: MenuItem[]
  }) => {
    // TODO(store/auth): setUserInfo 应持久化必要的用户和权限快照，或提供 restoreFromToken/restoreFromMe。
    // 只存在内存里会让刷新后的菜单和角色判断不稳定。
    user.value = data.user
    roles.value = data.roles
    permissions.value = data.permissions
    menus.value = data.menus
  }

  return {
    // State
    token,
    user,
    roles,
    permissions,
    menus,
    // Getters
    isLoggedIn,
    hasPermission,
    // Actions
    setToken,
    clearAuth,
    setUserInfo,
  }
})
