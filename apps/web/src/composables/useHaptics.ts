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

  function light() {
    trigger(25)
  }

  function medium() {
    trigger(50)
  }

  function heavy() {
    trigger(80)
  }

  function success() {
    trigger([30, 50, 30])
  }

  function error() {
    trigger([60, 80, 60, 80])
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
    light,
    medium,
    heavy,
    success,
    error,
    toggle,
  }
}
