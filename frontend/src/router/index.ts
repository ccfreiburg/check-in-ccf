import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/admin',
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
      component: () => import('../views/AdminParentDetailView.vue'),
      meta: { requiresAuth: true },
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
})

export default router
