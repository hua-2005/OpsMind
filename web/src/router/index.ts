// Vue Router：门户路由 + 后台路由 + 登录守卫
import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/auth/Login.vue'),
      meta: { guest: true },
    },
    {
      path: '/admin',
      component: () => import('@/layouts/AdminLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/admin/dashboard' },
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/admin/dashboard/Dashboard.vue'),
        },
        {
          path: 'accounts',
          name: 'Accounts',
          component: () => import('@/views/admin/accounts/AccountList.vue'),
        },
        {
          path: 'accounts/create',
          name: 'AccountCreate',
          component: () => import('@/views/admin/accounts/AccountForm.vue'),
        },
        {
          path: 'accounts/:id',
          name: 'AccountDetail',
          component: () => import('@/views/admin/accounts/AccountForm.vue'),
        },
        {
          path: 'roles',
          name: 'Roles',
          component: () => import('@/views/admin/roles/RoleList.vue'),
        },
      ],
    },
    {
      path: '/',
      redirect: '/admin',
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/admin',
    },
  ],
})

// 全局守卫
router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()

  // 恢复登录态
  if (!auth.isLoggedIn) {
    auth.restoreFromStorage()
  }

  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return next('/login')
  }

  if (to.path === '/login' && auth.isLoggedIn) {
    return next('/admin')
  }

  next()
})

export default router
