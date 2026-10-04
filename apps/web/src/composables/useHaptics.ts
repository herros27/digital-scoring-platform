import { ref } from 'vue'

export function useHaptics() {
  const isEnabled = ref(true)

  function trigger(pattern: number | number[] = 50) {
    if (!isEnabled.value) return
    if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
      try {
        navigator.vibrate(pattern)
      } catch {
        // Ignored if vibration not supported or blocked by browser permissions
      }
    }
  }

  function toggle() {
    isEnabled.value = !isEnabled.value
    if (isEnabled.value) {
      trigger(50)
    }
  }

  return {
    isEnabled,
    trigger,
    toggle,
  }
}
