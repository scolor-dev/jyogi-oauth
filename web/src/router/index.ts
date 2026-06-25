import { createRouter, createWebHistory } from 'vue-router'

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

export default router
