import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: () => {
        const parentToken = localStorage.getItem('parentToken')
        if (parentToken) return `/checkin/${parentToken}`
        return '/admin'
      },
    },
    {
      path: '/login',
      component: () => import('../views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/admin',
      component: () => import('../views/AdminChildListView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/parent/:id',
      name: 'parent-by-child',
      component: () => import('../views/AdminParentDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/parents/:id',
      name: 'parent-by-parent',
      component: () => import('../views/AdminParentDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/door',
      component: () => import('../views/AdminDoorView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/group',
      component: () => import('../views/AdminGroupView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/checkins/:id/notify',
      component: () => import('../views/AdminSendMessageView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/super',
      component: () => import('../views/AdminSuperView.vue'),
      meta: { requiresAuth: true, requiresSuperAdmin: true },
    },
    {
      path: '/checkin/:token',
      component: () => import('../views/ParentCheckinView.vue'),
      meta: { public: true },
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()

  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return '/login'
  }
  if (to.meta.requiresSuperAdmin && !auth.isSuperAdmin) {
    return '/admin'
  }
})

export default router
