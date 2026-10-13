<template>
  <div class="tablet-layout">
    <slot />
  </div>
</template>

<style scoped>
.tablet-layout {
  min-height: 100vh;
  min-height: 100dvh;
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
  overscroll-behavior: none;
}
</style>
