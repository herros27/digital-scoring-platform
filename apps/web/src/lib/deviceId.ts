import { generateUUID } from './uuid'

const KEY = 'ds_device_id'

/**
 * ID acak perangkat ini, dibuat sekali lalu disimpan di peramban. Dikirim saat login PIN (header X-Device-Id)
 * agar server menghitung PIN salah per tablet, bukan per IP: beberapa tablet sering berbagi satu IP (router,
 * Docker Desktop), dan satu juri yang salah ketik tidak boleh mengunci tablet lain. Ini bukan batas keamanan
 * (klien bisa menggantinya); server tetap punya batas per IP dan global.
 */
export function getDeviceId(): string {
  try {
    const existing = localStorage.getItem(KEY)
    if (existing && /^[A-Za-z0-9-]{8,64}$/.test(existing)) return existing
    const id = generateUUID()
    localStorage.setItem(KEY, id)
    return id
  } catch {
    // penyimpanan diblokir (mode privat): ID untuk sesi halaman ini saja
    return (memoryId ??= generateUUID())
  }
}

let memoryId: string | undefined
