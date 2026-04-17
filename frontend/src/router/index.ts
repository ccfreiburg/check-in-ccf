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
      name: 'first-registration',
      component: () => import('../views/FirstRegistrationView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/parent/:id',
      name: 'parent-by-child',
      component: () => import('../views/ParentDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/parents/:id',
      name: 'parent-by-parent',
      component: () => import('../views/ParentDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/guests/new',
      name: 'guest-new',
      component: () => import('../views/NewGuestView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/guests/:id/edit',
      name: 'guest-edit',
      component: () => import('../views/NewGuestView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/today',
      component: () => import('../views/ChildrenTodayView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/dashboard',
      component: () => import('../views/DashboardView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/tags',
      component: () => import('../views/TagHandoutView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/settings',
      component: () => import('../views/AdminView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/checkins/:id',
      component: () => import('../views/ChildDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/checkins/:id/notify',
      redirect: (to) => `/admin/checkins/${to.params.id}`,
    },
    // Legacy redirects
    { path: '/admin/door',  redirect: '/admin/today' },
    { path: '/admin/group', redirect: '/admin/today' },
    { path: '/admin/super', redirect: '/admin/today' },
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
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return '/admin/today'
  }
})

export default router
