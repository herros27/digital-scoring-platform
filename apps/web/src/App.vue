<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'
import KioskLayout from '@/layouts/KioskLayout.vue'
import TabletLayout from '@/layouts/TabletLayout.vue'

const layouts = { AdminLayout, KioskLayout, TabletLayout } as const

const route = useRoute()

const layoutComponent = computed(() => {
  const name = route.meta?.layout as keyof typeof layouts | undefined
  return name ? (layouts[name] ?? null) : null
})
</script>

<template>
  <component :is="layoutComponent" v-if="layoutComponent">
    <RouterView />
  </component>
  <RouterView v-else />
</template>
