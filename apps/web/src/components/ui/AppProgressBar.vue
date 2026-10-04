<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  value?: number
  max?: number
  variant?: 'primary' | 'secondary' | 'tertiary' | 'red' | 'blue' | 'emerald'
  height?: number
  showTrack?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  value: 0,
  max: 100,
  variant: 'primary',
  height: 6,
  showTrack: true,
})

const percentage = computed(() => {
  if (props.max <= 0) return 0
  return Math.min(100, Math.max(0, (props.value / props.max) * 100))
})
</script>

<template>
  <div
    class="app-progress-bar"
    :style="{ height: `${height}px` }"
    :class="{ 'app-progress-bar--no-track': !showTrack }"
  >
    <div
      class="app-progress-bar__fill"
      :class="`app-progress-bar__fill--${variant}`"
      :style="{ width: `${percentage}%` }"
    />
  </div>
</template>

<style scoped>
.app-progress-bar {
  width: 100%;
  background-color: rgba(255, 255, 255, 0.08);
  border-radius: var(--radius-full);
  overflow: hidden;
  position: relative;
}

.app-progress-bar--no-track {
  background-color: transparent;
}

.app-progress-bar__fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.3s ease;
}

/* Variants from Figma */
.app-progress-bar__fill--primary {
  background-color: var(--color-primary-peach);
  box-shadow: 0 0 10px rgba(252, 165, 165, 0.4);
}

.app-progress-bar__fill--red {
  background-color: var(--color-primary-600);
  box-shadow: 0 0 10px rgba(220, 38, 38, 0.5);
}

.app-progress-bar__fill--secondary,
.app-progress-bar__fill--blue {
  background-color: #93c5fd;
  box-shadow: 0 0 10px rgba(147, 197, 253, 0.4);
}

.app-progress-bar__fill--tertiary,
.app-progress-bar__fill--emerald {
  background-color: var(--color-tertiary-400);
  box-shadow: 0 0 10px rgba(52, 211, 153, 0.4);
}
</style>
