/** Label tampilan (Indonesia) untuk `result_type` hasil pertandingan dari server. */
export const RESULT_LABEL: Record<string, string> = {
  point_win: 'MENANG ANGKA (POINT WIN)',
  disqualification: 'MENANG DISKUALIFIKASI',
  referee_decision: 'KEPUTUSAN WASIT',
  tiebreak_penalties: 'MENANG ATURAN SERI (SANKSI TERENDAH)',
  tiebreak_technique: 'MENANG ATURAN SERI (TEKNIK TERBANYAK)',
  tiebreak_extra_round: 'MENANG BABAK TAMBAHAN',
  tiebreak_lighter_weight: 'MENANG ATURAN SERI (BERAT BADAN LEBIH RINGAN)',
  tiebreak_coin_toss: 'MENANG UNDIAN KOIN',
  draw: 'SERI (KEPUTUSAN WASIT)',
}

export const resultLabel = (type: string | undefined): string =>
  (type && RESULT_LABEL[type]) || type || RESULT_LABEL.point_win

/** Metode manual penentuan pemenang saat seri (pilihan wasit) dan labelnya. */
export const TIEBREAK_METHOD_LABEL: Record<string, string> = {
  referee_decision: 'Keputusan wasit',
  extra_round: 'Babak tambahan',
  lighter_weight: 'Pesilat lebih ringan (timbang ulang)',
  coin_toss: 'Undian lempar koin (Ketua Pertandingan)',
}
