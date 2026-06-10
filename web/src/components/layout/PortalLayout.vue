<template>
  <div class="portal-layout">
    <header class="portal-header">
      <div class="header-inner">
        <router-link to="/portal/chat" class="logo">OpsMind</router-link>
        <nav class="main-nav">
          <router-link to="/portal/chat" class="nav-link" active-class="nav-link--active">
            智能问答
          </router-link>
          <router-link to="/portal/tickets/submit" class="nav-link" active-class="nav-link--active">
            提交申告
          </router-link>
          <router-link to="/portal/tickets" class="nav-link" active-class="nav-link--active">
            我的申告
          </router-link>
          <router-link to="/portal/messages" class="nav-link" active-class="nav-link--active">
            消息
            <span v-if="unreadCount > 0" class="badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
          </router-link>
        </nav>
        <div class="header-right">
          <router-link to="/change-password" class="nav-link">修改密码</router-link>
          <button class="nav-link logout-btn" @click="handleLogout">退出</button>
        </div>
      </div>
    </header>
    <main class="portal-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { getUnreadCount } from '@/api/message'

const router = useRouter()
const authStore = useAuthStore()
const unreadCount = ref(0)

onMounted(async () => {
  try {
    const res = await getUnreadCount()
    const data = (res as any).data || res
    unreadCount.value = data?.count ?? data ?? 0
  } catch {
    // 静默失败
  }
})

function handleLogout() {
  authStore.clearAuth()
  router.push('/login')
}
</script>

<style scoped>
.portal-layout {
  min-height: 100vh;
  background: var(--bg-base);
  color: var(--text-primary);
  font-family: var(--font-family);
}

.portal-header {
  border-bottom: 1px solid var(--border-default);
  background: var(--bg-panel);
  position: sticky;
  top: 0;
  z-index: 50;
}

.header-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 32px;
}

.logo {
  font-size: 18px;
  font-weight: var(--font-weight-semibold, 600);
  color: var(--accent);
  text-decoration: none;
  flex-shrink: 0;
}

.main-nav {
  display: flex;
  gap: 4px;
  flex: 1;
}

.nav-link {
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 14px;
  padding: 8px 16px;
  border-radius: 6px;
  transition: color 0.15s, background 0.15s;
  position: relative;
}

.nav-link:hover {
  color: var(--text-primary);
  background: var(--bg-overlay);
}

.nav-link--active {
  color: var(--text-primary);
  background: var(--bg-overlay);
}

.header-right {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}

.logout-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  font-family: inherit;
}

.badge {
  position: absolute;
  top: 2px;
  right: 2px;
  background: var(--accent);
  color: #fff;
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
}

.portal-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 24px;
}
</style>
