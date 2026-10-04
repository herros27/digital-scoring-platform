<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'primary' | 'secondary' | 'tertiary' | 'neutral' | 'red' | 'blue' | 'emerald' | 'warning'
  size?: 'sm' | 'md' | 'lg'
  dot?: boolean
  pulse?: boolean
  outline?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'neutral',
  size: 'md',
  dot: false,
  pulse: false,
  outline: false,
})

const badgeClasses = computed(() => {
  return [
    'app-badge',
    `app-badge--${props.variant}`,
    `app-badge--${props.size}`,
    {
      'app-badge--outline': props.outline,
    },
  ]
})
</script>

<template>
  <span :class="badgeClasses">
    <span v-if="dot" class="app-badge__dot" :class="{ 'app-badge__dot--pulse': pulse }" />
    <slot />
  </span>
</template>

<style scoped>
.app-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-weight: 600;
  border-radius: var(--radius-full);
  white-space: nowrap;
  letter-spacing: -0.01em;
  transition: all var(--transition-fast);
}

/* Sizes */
.app-badge--sm {
  padding: 2px 8px;
  font-size: 0.6875rem;
}

.app-badge--md {
  padding: 4px 12px;
  font-size: 0.75rem;
}

.app-badge--lg {
  padding: 6px 16px;
  font-size: 0.875rem;
}

/* Variants */
.app-badge--primary {
  background-color: rgba(252, 165, 165, 0.15);
  color: var(--color-primary-peach);
  border: 1px solid rgba(252, 165, 165, 0.3);
}

.app-badge--red {
  background-color: rgba(220, 38, 38, 0.18);
  color: #f87171;
  border: 1px solid rgba(220, 38, 38, 0.4);
}

.app-badge--blue {
  background-color: rgba(37, 99, 235, 0.18);
  color: #60a5fa;
  border: 1px solid rgba(37, 99, 235, 0.4);
}

.app-badge--emerald,
.app-badge--tertiary {
  background-color: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.35);
}

.app-badge--neutral {
  background-color: rgba(255, 255, 255, 0.06);
  color: var(--color-neutral-300);
  border: 1px solid var(--border-subtle);
}

.app-badge--warning {
  background-color: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

/* Dot */
.app-badge__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: currentColor;
}

.app-badge__dot--pulse {
  animation: pulse 1.8s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.4;
    transform: scale(0.85);
  }
}
</style>
