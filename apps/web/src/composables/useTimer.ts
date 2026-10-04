import { ref, computed, onUnmounted } from 'vue'

export function useTimer() {
  const remainingMs = ref(0)
  const isRunning = ref(false)
  let ticker: any = null

  function sync(ms: number, running: boolean) {
    remainingMs.value = Math.max(0, ms)
    isRunning.value = running

    if (ticker) {
      clearInterval(ticker)
      ticker = null
    }

    if (running && remainingMs.value > 0) {
      ticker = setInterval(() => {
        if (remainingMs.value > 100) {
          remainingMs.value -= 100
        } else {
          remainingMs.value = 0
          isRunning.value = false
          if (ticker) {
            clearInterval(ticker)
            ticker = null
          }
        }
      }, 100)
    }
  }

  function pause() {
    isRunning.value = false
    if (ticker) {
      clearInterval(ticker)
      ticker = null
    }
  }

  const minutes = computed(() => {
    const totalSecs = Math.max(0, Math.floor(remainingMs.value / 1000))
    return Math.floor(totalSecs / 60)
  })

  const seconds = computed(() => {
    const totalSecs = Math.max(0, Math.floor(remainingMs.value / 1000))
    return totalSecs % 60
  })

  const formattedTime = computed(() => {
    return `${String(minutes.value).padStart(2, '0')}:${String(seconds.value).padStart(2, '0')}`
  })

  onUnmounted(() => {
    if (ticker) clearInterval(ticker)
  })

  return {
    remainingMs,
    isRunning,
    minutes,
    seconds,
    formattedTime,
    sync,
    pause,
  }
}
