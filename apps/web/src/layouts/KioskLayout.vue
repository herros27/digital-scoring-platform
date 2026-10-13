<template>
  <div class="kiosk-layout">
    <slot />
  </div>
</template>

<style scoped>
.kiosk-layout {
  min-height: 100vh;
  min-height: 100dvh;
  overflow: hidden;
  user-select: none;
}
</style>
