import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  // TODO(store/app): sidebarCollapsed 和 unreadMessageCount 没有持久化/初始化。
  // 刷新后侧边栏偏好丢失，未读数也需要从 getUnreadCount 拉取。
  // State
  const sidebarCollapsed = ref(false)
  const unreadMessageCount = ref(0)

  // Actions
  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  const setUnreadMessageCount = (count: number) => {
    unreadMessageCount.value = count
  }

  return {
    sidebarCollapsed,
    unreadMessageCount,
    toggleSidebar,
    setUnreadMessageCount,
  }
})
