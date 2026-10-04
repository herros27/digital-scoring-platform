<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'

const route = useRoute()

const layoutComponent = computed(() => {
  if (route.meta?.layout === 'AdminLayout') {
    return AdminLayout
  }
  return null
})
</script>

<template>
  <component :is="layoutComponent" v-if="layoutComponent">
    <RouterView />
  </component>
  <RouterView v-else />
</template>
