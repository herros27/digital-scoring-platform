let ctx: AudioContext | null = null

/**
 * Bunyikan gong sintetis (Web Audio). Browser memblokir suara sebelum ada interaksi pengguna
 * pada halaman; kegagalan diabaikan agar tidak mengganggu alur pertandingan.
 */
export function playGong(seconds = 2.6): void {
  try {
    const Ctor = window.AudioContext
    if (!Ctor) return
    ctx ??= new Ctor()
    if (ctx.state === 'suspended') void ctx.resume()

    const now = ctx.currentTime
    const master = ctx.createGain()
    master.gain.setValueAtTime(0.0001, now)
    master.gain.exponentialRampToValueAtTime(0.7, now + 0.01)
    master.gain.exponentialRampToValueAtTime(0.0001, now + seconds)
    master.connect(ctx.destination)

    // Gong = nada dasar dengan overtone tidak harmonis yang meluruh pelan
    for (const [freq, gain] of [
      [196, 1],
      [293, 0.6],
      [412, 0.45],
      [587, 0.3],
    ] as const) {
      const osc = ctx.createOscillator()
      const g = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.setValueAtTime(freq, now)
      g.gain.setValueAtTime(gain, now)
      osc.connect(g)
      g.connect(master)
      osc.start(now)
      osc.stop(now + seconds + 0.1)
    }
  } catch {
    // suara tidak tersedia
  }
}
