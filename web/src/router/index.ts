import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import type { Role } from '../types'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    roles?: Role[]
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('../views/Home.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/Login.vue'),
    },
    {
      path: '/consent',
      name: 'consent',
      component: () => import('../views/Consent.vue'),
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('../views/Dashboard.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/change-password',
      name: 'change-password',
      component: () => import('../views/ChangePassword.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('../views/admin/Admin.vue'),
      meta: { requiresAuth: true, roles: ['admin', 'moderator'] },
      children: [
        {
          path: '',
          name: 'admin-home',
          component: () => import('../views/admin/AdminHome.vue'),
        },
        {
          path: 'members',
          name: 'admin-members',
          component: () => import('../views/admin/AdminMembers.vue'),
        },
        {
          path: 'clients',
          name: 'admin-clients',
          component: () => import('../views/admin/AdminClients.vue'),
          meta: { roles: ['admin'] },
        },
        {
          path: 'scopes',
          name: 'admin-scopes',
          component: () => import('../views/admin/AdminScopes.vue'),
          meta: { roles: ['admin'] },
        },
        {
          path: 'audit-logs',
          name: 'admin-audit-logs',
          component: () => import('../views/admin/AdminAuditLogs.vue'),
        },
      ],
    },
  ],
})

const allowedWhileForceChange = ['change-password', 'login', 'home']

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.loaded) await auth.fetchMe()

  if (to.meta.requiresAuth && !auth.isLoggedIn()) {
    return { name: 'login' }
  }

  if (auth.isLoggedIn() && auth.mustChangePassword()) {
    if (!allowedWhileForceChange.includes(to.name as string)) {
      return { name: 'change-password' }
    }
  }

  if (to.meta.roles && (!auth.member || !to.meta.roles.includes(auth.member.role))) {
    return { name: 'dashboard' }
  }
})

export default router
