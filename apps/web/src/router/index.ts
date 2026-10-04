import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    redirect: '/login',
  },
  {
    path: '/design-system',
    name: 'design-system',
    component: () => import('../views/DesignSystemView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/scoreboard/:arena_id?',
    name: 'scoreboard',
    component: () => import('../views/scoreboard/ScoreboardView.vue'),
    meta: { requiresAuth: false, layout: 'KioskLayout' },
  },
  {
    path: '/judge',
    name: 'judge',
    component: () => import('../views/judge/JudgeScoringView.vue'),
    meta: { requiresAuth: true, allowedRoles: ['judge', 'admin'], layout: 'TabletLayout' },
  },
  {
    path: '/referee',
    name: 'referee',
    component: () => import('../views/referee/RefereeConsoleView.vue'),
    meta: { requiresAuth: true, allowedRoles: ['wasit', 'admin', 'dewan'] },
  },
  {
    path: '/ops',
    name: 'operations panel',
    component: () => import('../views/ops/OpsShellView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/auth/LoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('../views/auth/RegisterView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/judge/login',
    name: 'judge-login',
    component: () => import('../views/judge/JudgeLoginView.vue'),
    meta: { requiresAuth: false, layout: 'TabletLayout' },
  },
  {
    path: '/auth',
    redirect: '/login',
  },
  {
    path: '/admin',
    redirect: '/admin/competitions',
  },
  {
    path: '/admin/pins',
    name: 'admin-pins',
    component: () => import('../views/admin/DevicePinManager.vue'),
    meta: { requiresAuth: true, allowedRoles: ['admin', 'dewan'], layout: 'AdminLayout' },
  },
  {
    path: '/admin/competitions',
    name: 'admin-competitions',
    component: () => import('../views/admin/CompetitionsView.vue'),
    meta: { requiresAuth: true, allowedRoles: ['admin', 'dewan'], layout: 'AdminLayout' },
  },
  {
    path: '/competitions',
    redirect: '/admin/competitions',
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// RBAC & Auth Navigation Guard
router.beforeEach((to, _from, next) => {
  const token = sessionStorage.getItem('ds_token')
  const requiresAuth = to.meta.requiresAuth === true

  if (requiresAuth && !token) {
    return next({ path: '/login', query: { redirect: to.fullPath } })
  }

  next()
})

export default router
