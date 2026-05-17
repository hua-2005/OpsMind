<script setup lang="ts">
// 后台管理布局：侧边栏菜单 + 顶栏 + 内容区
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import type { MenuItem } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

// 登录后加载权限树
if (auth.isLoggedIn && auth.menus.length === 0) {
  auth.loadPermissions()
}

const activeMenu = computed(() => {
  const p = route.path.replace('/admin/', '')
  return p.split('/')[0] || ''
})

function goMenu(item: MenuItem) {
  if (item.path) {
    router.push(item.path)
  }
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="admin-layout">
    <aside class="admin-sidebar">
      <div class="sidebar-brand">OpsMind</div>
      <nav class="sidebar-nav">
        <div
          v-for="item in auth.menus"
          :key="item.code"
          class="nav-item"
          :class="{ active: activeMenu === item.code }"
          @click="goMenu(item)"
        >
          {{ item.name }}
        </div>
      </nav>
      <div class="sidebar-footer">
        <span>{{ auth.user?.real_name }}</span>
        <button class="logout-btn" @click="handleLogout">退出</button>
      </div>
    </aside>
    <main class="admin-main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  min-height: 100vh;
  background: #0f1011;
}
.admin-sidebar {
  width: 220px;
  background: #191a1b;
  border-right: 1px solid rgba(255,255,255,0.05);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}
.sidebar-brand {
  padding: 20px 16px;
  font-size: 18px;
  font-weight: 590;
  letter-spacing: -0.5px;
  color: #f7f8f8;
  border-bottom: 1px solid rgba(255,255,255,0.05);
}
.sidebar-nav {
  flex: 1;
  padding: 8px 0;
}
.nav-item {
  padding: 10px 16px;
  color: #8a8f98;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.15s;
}
.nav-item:hover {
  color: #d0d6e0;
  background: rgba(255,255,255,0.03);
}
.nav-item.active {
  color: #f7f8f8;
  background: rgba(255,255,255,0.06);
}
.sidebar-footer {
  padding: 12px 16px;
  border-top: 1px solid rgba(255,255,255,0.05);
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #8a8f98;
  font-size: 13px;
}
.logout-btn {
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.08);
  color: #d0d6e0;
  padding: 4px 10px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
}
.admin-main {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}
</style>
