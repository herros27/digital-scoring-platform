/**
 * Centralized API Endpoints Registry for DigitalScoring REST Services
 * 100% Complete Mapping matching OpenAPI 3.0 Documentation (docs/openapi.yaml)
 */
export const ENDPOINTS = {
  // Bagan publik (penonton, tanpa login) — hanya kejuaraan aktif, hanya nama & kontingen atlet
  PUBLIC: {
    COMPETITIONS: '/api/v1/public/competitions',
    BRACKETS: (compId: string) => `/api/v1/public/competitions/${compId}/brackets`,
    BRACKET: (compId: string, categoryId: string) =>
      `/api/v1/public/competitions/${compId}/brackets/${categoryId}`,
  },

  // Authentication
  AUTH: {
    OFFICIAL_LOGIN: '/api/v1/auth/login',
    JUDGE_LOGIN: '/api/v1/auth/judge-login',
    CONTINGENT_REGISTER: '/api/v1/auth/contingent/register',
    CONTINGENT_LOGIN: '/api/v1/auth/contingent/login',
    LOGOUT: '/api/v1/auth/logout',
    CHANGE_PASSWORD: '/api/v1/auth/change-password',
    CREATE_PIN: '/api/v1/auth/create-pin',
    PINS: '/api/v1/auth/pins',
  },

  // Competitions
  COMPETITIONS: {
    LIST: '/api/v1/competitions',
    STATS: '/api/v1/competitions/stats',
    CREATE: '/api/v1/competitions',
    DETAIL: (id: string) => `/api/v1/competitions/${id}`,
    UPDATE: (id: string) => `/api/v1/competitions/${id}`,
    ACTIVATE: (id: string) => `/api/v1/competitions/${id}/activate`,
    PAUSE: (id: string) => `/api/v1/competitions/${id}/pause`,
    FINISH: (id: string) => `/api/v1/competitions/${id}/finish`,
    CANCEL: (id: string) => `/api/v1/competitions/${id}/cancel`,
    CATEGORIES: (compId: string) => `/api/v1/competitions/${compId}/categories`,
    ARENAS: (compId: string) => `/api/v1/competitions/${compId}/arenas`,
    MATCHES: (compId: string) => `/api/v1/competitions/${compId}/matches`,
    BRACKETS: (compId: string) => `/api/v1/competitions/${compId}/brackets`,
  },

  // Teknik skor menurut ruleset kejuaraan (layar juri sebelum ada partai dimuat)
  TECHNIQUES: (compId: string) => `/api/v1/competitions/${compId}/techniques`,

  // Categories
  CATEGORIES: {
    LIST: (compId: string) => `/api/v1/competitions/${compId}/categories`,
    CREATE: (compId: string) => `/api/v1/competitions/${compId}/categories`,
    RESET: (compId: string) => `/api/v1/competitions/${compId}/categories`,
    HOME_ARENA: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/categories/${id}/home-arena`,
    BRACKET_LOCK: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/categories/${id}/bracket-lock`,
    ASSIGN_RULESETS: (compId: string) => `/api/v1/competitions/${compId}/categories/ruleset`,
    DETAIL: (compId: string, id: string) => `/api/v1/competitions/${compId}/categories/${id}`,
  },

  // Athletes & Participants
  ATHLETES: {
    LIST: '/api/v1/athletes',
    CREATE: '/api/v1/athletes',
    DETAIL: (id: string) => `/api/v1/athletes/${id}`,
    BY_COMPETITION: (compId: string) => `/api/v1/competitions/${compId}/athletes`,
    REGISTER_TO_COMP: (compId: string) => `/api/v1/competitions/${compId}/athletes`,
    PARTICIPANTS: (compId: string) => `/api/v1/competitions/${compId}/participants`,
    PARTICIPANT: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/participants/${id}`,
    VERIFY_PARTICIPANT: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/participants/${id}/verify`,
    PARTICIPANT_CATEGORY: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/participants/${id}/category`,
  },

  // Contingents
  CONTINGENTS: {
    LIST: '/api/v1/contingents',
    DETAIL: (id: string) => `/api/v1/contingents/${id}`,
    PASSWORD: (id: string) => `/api/v1/contingents/${id}/password`,
  },

  // Arenas
  ARENAS: {
    LIST: (compId: string) => `/api/v1/competitions/${compId}/arenas`,
    CREATE: (compId: string) => `/api/v1/competitions/${compId}/arenas`,
    DETAIL: (compId: string, id: string) => `/api/v1/competitions/${compId}/arenas/${id}`,
    PRESENCE: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/arenas/${id}/presence`,
    UPDATE: (compId: string, id: string) => `/api/v1/competitions/${compId}/arenas/${id}`,
    DELETE: (compId: string, id: string) => `/api/v1/competitions/${compId}/arenas/${id}`,
  },

  // Matches
  MATCHES: {
    LIST: (compId: string) => `/api/v1/competitions/${compId}/matches`,
    DETAIL: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}`,
    CREATE: (compId: string) => `/api/v1/competitions/${compId}/matches`,
    LOAD_ARENA: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/matches/${id}/load-arena`,
    START: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/start`,
    PAUSE: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/pause`,
    RESUME: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/resume`,
    FINISH: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/finish`,
    CANCEL: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/cancel`,
    OUTCOME: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/outcome`,
    RESCHEDULE: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/matches/${id}/reschedule`,
    SCORES: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/scores`,
    SUBMIT_SCORE: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/matches/${id}/scores`,
    EVENTS: (compId: string, id: string) => `/api/v1/competitions/${compId}/matches/${id}/events`,
  },

  // Brackets
  BRACKETS: {
    TREE: (compId: string) => `/api/v1/competitions/${compId}/brackets`,
    GENERATE: (compId: string) => `/api/v1/competitions/${compId}/brackets/generate`,
    RESET: (compId: string, categoryId: string) =>
      `/api/v1/competitions/${compId}/categories/${categoryId}/bracket`,
  },

  // Participants
  PARTICIPANTS: {
    LIST: (compId: string) => `/api/v1/competitions/${compId}/participants`,
    VERIFY: (compId: string, id: string) =>
      `/api/v1/competitions/${compId}/participants/${id}/verify`,
  },

  // Rulesets
  RULESETS: {
    LIST: '/api/v1/rulesets',
    CREATE: '/api/v1/rulesets',
    DETAIL: (id: string) => `/api/v1/rulesets/${id}`,
    UPDATE: (id: string) => `/api/v1/rulesets/${id}`,
    DELETE: (id: string) => `/api/v1/rulesets/${id}`,
  },

  // Officials
  OFFICIALS: {
    LIST: '/api/v1/officials',
    CREATE: '/api/v1/officials',
    DETAIL: (id: string) => `/api/v1/officials/${id}`,
    UPDATE: (id: string) => `/api/v1/officials/${id}`,
    DELETE: (id: string) => `/api/v1/officials/${id}`,
    PASSWORD: (id: string) => `/api/v1/officials/${id}/password`,
  },

  // Reports
  REPORTS: {
    SUMMARY: (compId: string) => `/api/v1/competitions/${compId}/report/summary`,
    MEDALS: (compId: string) => `/api/v1/competitions/${compId}/report/medals`,
    EXPORT: (compId: string) => `/api/v1/competitions/${compId}/report/export`,
  },

  // Live Displays & Telemetry
  LIVE: {
    ARENAS: '/api/v1/live/arenas',
    SCHEDULE: '/api/v1/live/schedule',
    SCOREBOARD: (arenaId: string) => `/api/v1/live/scoreboard/${arenaId}`,
  },

  // System Health
  SYSTEM: {
    HEALTH: '/api/v1/health',
  },
} as const
