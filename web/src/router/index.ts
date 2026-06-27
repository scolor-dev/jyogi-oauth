import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

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
    },
    {
      path: '/change-password',
      name: 'change-password',
      component: () => import('../views/ChangePassword.vue'),
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('../views/admin/Admin.vue'),
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
        },
      ],
    },
  ],
})

const allowedWhileForceChange = ['change-password', 'login', 'home']

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.loaded) await auth.fetchMe()

  if (auth.isLoggedIn() && auth.mustChangePassword()) {
    if (!allowedWhileForceChange.includes(to.name as string)) {
      return { name: 'change-password' }
    }
  }
})

export default router
