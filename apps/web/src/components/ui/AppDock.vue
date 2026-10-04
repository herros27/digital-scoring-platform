<script setup lang="ts">
defineProps<{
  activeId?: string
}>()

const emit = defineEmits<{
  (e: 'select', id: string): void
}>()
</script>

<template>
  <nav class="app-dock">
    <button
      class="app-dock__item"
      :class="{ 'app-dock__item--active': activeId === 'home' }"
      title="Home"
      @click="emit('select', 'home')"
    >
      <slot name="home">
        <!-- Home SVG -->
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
          <polyline points="9 22 9 12 15 12 15 22" />
        </svg>
      </slot>
    </button>

    <button
      class="app-dock__item"
      :class="{ 'app-dock__item--active': activeId === 'search' }"
      title="Search"
      @click="emit('select', 'search')"
    >
      <slot name="search">
        <!-- Search SVG -->
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
      </slot>
    </button>

    <button
      class="app-dock__item"
      :class="{ 'app-dock__item--active': activeId === 'user' }"
      title="Profile"
      @click="emit('select', 'user')"
    >
      <slot name="user">
        <!-- User SVG -->
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
          <circle cx="12" cy="7" r="4" />
        </svg>
      </slot>
    </button>
  </nav>
</template>

<style scoped>
.app-dock {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  background-color: var(--color-neutral-850);
  border: 1px solid var(--color-neutral-700);
  border-radius: var(--radius-full);
  padding: 6px 10px;
  box-shadow: var(--shadow-lg);
}

.app-dock__item {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: none;
  color: var(--color-neutral-400);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.app-dock__item:hover {
  color: var(--text-main);
  background-color: rgba(255, 255, 255, 0.05);
}

/* Active item matching Figma dock with peach background */
.app-dock__item--active {
  background-color: var(--color-primary-peach) !important;
  color: #0f172a !important;
  box-shadow: 0 2px 8px rgba(252, 165, 165, 0.3);
}
</style>
