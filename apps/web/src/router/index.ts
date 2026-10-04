import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('../views/HomeView.vue'),
    },
    {
      path: '/design-system',
      name: 'design-system',
      component: () => import('../views/DesignSystemView.vue'),
    },
    {
      path: '/scoreboard/:arena_id?',
      name: 'scoreboard',
      component: () => import('../views/scoreboard/ScoreboardView.vue'),
    },
    {
      path: '/judge',
      name: 'judge',
      component: () => import('../views/judge/JudgeScoringView.vue'),
    },
    {
      path: '/referee',
      name: 'referee',
      component: () => import('../views/referee/RefereeConsoleView.vue'),
    },
    {
      path: '/bracket',
      name: 'bracket',
      component: () => import('../views/bracket/BracketOpsView.vue'),
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('../views/bracket/BracketOpsView.vue'),
    },
    {
      path: '/admin/pins',
      name: 'admin-pins',
      component: () => import('../views/bracket/BracketOpsView.vue'),
    },
  ],
})

export default router
