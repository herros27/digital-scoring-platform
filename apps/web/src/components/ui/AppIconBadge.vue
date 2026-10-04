<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'peach' | 'blue' | 'emerald' | 'red' | 'neutral'
  size?: 'sm' | 'md' | 'lg'
  clickable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'peach',
  size: 'md',
  clickable: false,
})

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()

const classes = computed(() => [
  'app-icon-badge',
  `app-icon-badge--${props.variant}`,
  `app-icon-badge--${props.size}`,
  { 'app-icon-badge--clickable': props.clickable },
])
</script>

<template>
  <div :class="classes" @click="clickable && emit('click', $event)">
    <slot />
  </div>
</template>

<style scoped>
.app-icon-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-xs);
  transition: all var(--transition-fast);
  user-select: none;
  flex-shrink: 0;
}

/* Sizes */
.app-icon-badge--sm {
  width: 28px;
  height: 28px;
}
.app-icon-badge--md {
  width: 36px;
  height: 36px;
}
.app-icon-badge--lg {
  width: 44px;
  height: 44px;
}

/* Colors matching Figma */
.app-icon-badge--peach {
  background-color: var(--color-primary-peach);
  color: #0f172a;
}

.app-icon-badge--blue {
  background-color: #93c5fd;
  color: #0f172a;
}

.app-icon-badge--emerald {
  background-color: var(--color-tertiary-400);
  color: #022c22;
}

.app-icon-badge--red {
  background-color: #fca5a5;
  color: #7f1d1d;
}

.app-icon-badge--neutral {
  background-color: var(--color-neutral-800);
  color: var(--color-neutral-200);
  border: 1px solid var(--color-neutral-700);
}

.app-icon-badge--clickable {
  cursor: pointer;
}
.app-icon-badge--clickable:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}
.app-icon-badge--clickable:active {
  transform: translateY(0);
}
</style>
