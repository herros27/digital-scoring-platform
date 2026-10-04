<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'default' | 'elevated' | 'red-corner' | 'blue-corner' | 'emerald'
  glow?: boolean
  padding?: 'none' | 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  glow: false,
  padding: 'md',
})

const cardClasses = computed(() => [
  'app-card',
  `app-card--${props.variant}`,
  `app-card--pad-${props.padding}`,
  { 'app-card--glow': props.glow },
])
</script>

<template>
  <div :class="cardClasses">
    <div v-if="$slots.header" class="app-card__header">
      <slot name="header" />
    </div>
    <div class="app-card__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="app-card__footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.app-card {
  background-color: var(--bg-card);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-lg);
  transition: all var(--transition-normal);
  position: relative;
  overflow: hidden;
}

/* Padding options */
.app-card--pad-none .app-card__body {
  padding: 0;
}
.app-card--pad-sm .app-card__body {
  padding: 12px;
}
.app-card--pad-md .app-card__body {
  padding: 20px;
}
.app-card--pad-lg .app-card__body {
  padding: 28px;
}

.app-card__header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.app-card__footer {
  padding: 14px 20px;
  border-top: 1px solid var(--border-subtle);
  background-color: rgba(0, 0, 0, 0.15);
}

/* Variants */
.app-card--elevated {
  background-color: var(--bg-card-elevated);
  box-shadow: var(--shadow-lg);
}

.app-card--red-corner {
  border-color: rgba(220, 38, 38, 0.4);
  background: linear-gradient(180deg, rgba(220, 38, 38, 0.08) 0%, rgba(19, 29, 51, 0.95) 100%);
}
.app-card--red-corner.app-card--glow {
  box-shadow: var(--shadow-glow-red);
}

.app-card--blue-corner {
  border-color: rgba(37, 99, 235, 0.4);
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.08) 0%, rgba(19, 29, 51, 0.95) 100%);
}
.app-card--blue-corner.app-card--glow {
  box-shadow: var(--shadow-glow-blue);
}

.app-card--emerald {
  border-color: rgba(16, 185, 129, 0.4);
  background: linear-gradient(180deg, rgba(16, 185, 129, 0.08) 0%, rgba(19, 29, 51, 0.95) 100%);
}
.app-card--emerald.app-card--glow {
  box-shadow: var(--shadow-glow-green);
}
</style>
