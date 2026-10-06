-- +goose Up
-- Ruleset kini memuat tabel sanksi, daftar teknik, dan aturan konsensus juri (lihat internal/rules).
-- Preset bawaan yang masih berformat lama dilengkapi: Teguran II = -2, Peringatan III = diskualifikasi,
-- Tangkapan + Jatuhan = 4, konsensus 2 dari 3 juri dalam 3 detik. Ruleset buatan pengguna tidak disentuh
-- (bila bagian ini kosong, server memakai nilai turunan/bawaan yang sama).
UPDATE rulesets
SET scoring = scoring || '{
  "techniques": [
    {"type": "technique",  "label": "Pukulan",             "points": 1},
    {"type": "knockdown",  "label": "Tendangan",           "points": 2},
    {"type": "fall",       "label": "Jatuhan",             "points": 3},
    {"type": "catch_fall", "label": "Tangkapan + Jatuhan", "points": 4}
  ],
  "penalties": [
    {"type": "binaan_1",     "label": "Binaan 1",     "points": 0},
    {"type": "binaan_2",     "label": "Binaan 2",     "points": 0},
    {"type": "teguran_1",    "label": "Teguran 1",    "points": 1},
    {"type": "teguran_2",    "label": "Teguran 2",    "points": 2},
    {"type": "peringatan_1", "label": "Peringatan 1", "points": 5},
    {"type": "peringatan_2", "label": "Peringatan 2", "points": 10},
    {"type": "peringatan_3", "label": "Peringatan 3", "points": 0, "action": "disqualify"}
  ],
  "consensus": {"judges": 3, "majority": 2, "window_ms": 3000}
}'::jsonb
WHERE name IN ('IPSI 2024', 'PERSILAT International')
  AND NOT (scoring ? 'techniques');

-- +goose Down
UPDATE rulesets
SET scoring = scoring - 'techniques' - 'penalties' - 'consensus'
WHERE name IN ('IPSI 2024', 'PERSILAT International');
