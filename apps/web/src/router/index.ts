import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    redirect: '/scoreboard',
  },
  {
    path: '/scoreboard',
    name: 'scoreboard-list',
    component: () => import('../views/scoreboard/ScoreboardListView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/scoreboard/:arena_id',
    name: 'scoreboard',
    component: () => import('../views/scoreboard/ScoreboardView.vue'),
    meta: { requiresAuth: false, layout: 'KioskLayout' },
  },
  {
    path: '/bagan',
    name: 'bagan-list',
    component: () => import('../views/bagan/BaganListView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/bagan/:comp_id/:category_id',
    name: 'bagan',
    component: () => import('../views/bagan/BaganView.vue'),
    meta: { requiresAuth: false },
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
    meta: { requiresAuth: true, allowedRoles: ['wasit', 'referee', 'admin', 'dewan'] },
  },
  {
    path: '/ops',
    name: 'operations panel',
    component: () => import('../views/ops/OpsShellView.vue'),
    meta: { requiresAuth: true, allowedRoles: ['admin', 'dewan', 'operator', 'contingent'] },
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
    redirect: '/ops?tab=kompetisi',
  },
  {
    path: '/admin/competitions',
    redirect: '/ops?tab=kompetisi',
  },
  {
    path: '/admin/pins',
    redirect: '/ops?tab=pin',
  },
  {
    path: '/competitions',
    redirect: '/ops?tab=kompetisi',
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// Halaman tujuan default per role ketika akses ke rute ditolak
function homeForRole(role: string): string {
  switch (role) {
    case 'judge':
      return '/judge'
    case 'wasit':
    case 'referee':
      return '/referee'
    case 'admin':
    case 'dewan':
    case 'operator':
    case 'contingent':
      return '/ops'
    default:
      return '/login'
  }
}

// RBAC & Auth Navigation Guard
router.beforeEach((to) => {
  const token = sessionStorage.getItem('ds_token')
  const requiresAuth = to.meta.requiresAuth === true

  if (requiresAuth && !token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  const allowedRoles = to.meta.allowedRoles as string[] | undefined
  if (requiresAuth && allowedRoles?.length) {
    let role = ''
    try {
      role = JSON.parse(sessionStorage.getItem('ds_auth_session') || 'null')?.role || ''
    } catch {
      // sesi rusak: perlakukan sebagai tidak punya role
    }
    if (!allowedRoles.includes(role)) {
      const fallback = homeForRole(role)
      // Hindari redirect loop bila halaman fallback juga tidak diizinkan
      return fallback === to.path ? '/login' : fallback
    }
  }
})

export default router
