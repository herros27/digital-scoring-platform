<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'primary' | 'secondary' | 'inverted' | 'outlined' | 'corner-red' | 'corner-blue' | 'tertiary' | 'emerald' | 'danger'
  size?: 'sm' | 'md' | 'lg' | 'xl'
  disabled?: boolean
  loading?: boolean
  fullWidth?: boolean
  type?: 'button' | 'submit' | 'reset'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
  fullWidth: false,
  type: 'button',
})

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()

function handleClick(event: MouseEvent) {
  if (!props.disabled && !props.loading) {
    emit('click', event)
  }
}

const buttonClasses = computed(() => {
  return [
    'app-btn',
    `app-btn--${props.variant}`,
    `app-btn--${props.size}`,
    {
      'app-btn--disabled': props.disabled || props.loading,
      'app-btn--full': props.fullWidth,
    },
  ]
})
</script>

<template>
  <button
    :type="type"
    :class="buttonClasses"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <span v-if="loading" class="app-btn__spinner" />
    <span class="app-btn__content">
      <slot />
    </span>
  </button>
</template>

<style scoped>
.app-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-family: var(--font-sans);
  font-weight: 600;
  border-radius: var(--radius-sm);
  cursor: pointer;
  border: 1px solid transparent;
  transition: all var(--transition-fast);
  text-decoration: none;
  user-select: none;
  position: relative;
  letter-spacing: -0.01em;
}

.app-btn--full {
  width: 100%;
}

/* --- Sizes --- */
.app-btn--sm {
  padding: 6px 14px;
  font-size: 0.8125rem;
  border-radius: var(--radius-xs);
}

.app-btn--md {
  padding: 10px 20px;
  font-size: 0.9375rem;
  border-radius: var(--radius-sm);
}

.app-btn--lg {
  padding: 14px 28px;
  font-size: 1.0625rem;
  border-radius: var(--radius-md);
}

.app-btn--xl {
  padding: 18px 36px;
  font-size: 1.25rem;
  font-weight: 700;
  border-radius: var(--radius-lg);
}

/* --- Variants from Figma (00_design_system.png) --- */

/* Primary: Peach/Coral tint as shown in Figma */
.app-btn--primary {
  background-color: var(--color-primary-peach);
  color: #0f172a;
  border-color: transparent;
}
.app-btn--primary:hover:not(:disabled) {
  background-color: #fca5a5;
  filter: brightness(1.05);
  transform: translateY(-1px);
}
.app-btn--primary:active:not(:disabled) {
  transform: translateY(0);
}

/* Secondary: Dark slate with subtle border */
.app-btn--secondary {
  background-color: var(--color-neutral-800);
  color: var(--color-neutral-100);
  border-color: var(--color-neutral-700);
}
.app-btn--secondary:hover:not(:disabled) {
  background-color: var(--color-neutral-700);
  border-color: var(--color-neutral-600);
}

/* Inverted: Clean white with dark text */
.app-btn--inverted {
  background-color: #ffffff;
  color: #0f172a;
  border-color: #ffffff;
}
.app-btn--inverted:hover:not(:disabled) {
  background-color: #f1f5f9;
}

/* Outlined: Dark with outlined border */
.app-btn--outlined {
  background-color: transparent;
  color: var(--color-primary-peach);
  border-color: var(--color-neutral-700);
}
.app-btn--outlined:hover:not(:disabled) {
  background-color: rgba(255, 255, 255, 0.05);
  border-color: var(--color-primary-peach);
}

/* Corner Red (Pencak Silat Sudut Merah) */
.app-btn--corner-red {
  background-color: var(--color-primary-600);
  color: #ffffff;
  box-shadow: 0 4px 14px rgba(220, 38, 38, 0.35);
}
.app-btn--corner-red:hover:not(:disabled) {
  background-color: var(--color-primary-700);
  box-shadow: 0 6px 20px rgba(220, 38, 38, 0.5);
  transform: translateY(-1px);
}

/* Corner Blue (Pencak Silat Sudut Biru) */
.app-btn--corner-blue {
  background-color: var(--color-secondary-600);
  color: #ffffff;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.35);
}
.app-btn--corner-blue:hover:not(:disabled) {
  background-color: var(--color-secondary-700);
  box-shadow: 0 6px 20px rgba(37, 99, 235, 0.5);
  transform: translateY(-1px);
}

/* Tertiary / Emerald */
.app-btn--tertiary,
.app-btn--emerald {
  background-color: var(--color-tertiary-500);
  color: #022c22;
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
}
.app-btn--tertiary:hover:not(:disabled),
.app-btn--emerald:hover:not(:disabled) {
  background-color: var(--color-tertiary-400);
  transform: translateY(-1px);
}

/* Danger / Destructive Red */
.app-btn--danger {
  background-color: #DC2626;
  color: #FFFFFF;
  box-shadow: 0 4px 14px rgba(220, 38, 38, 0.35);
}
.app-btn--danger:hover:not(:disabled) {
  background-color: #EF4444;
  transform: translateY(-1px);
}

/* Disabled State */
.app-btn--disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none !important;
  box-shadow: none !important;
}

/* Spinner */
.app-btn__spinner {
  width: 16px;
  height: 16px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
