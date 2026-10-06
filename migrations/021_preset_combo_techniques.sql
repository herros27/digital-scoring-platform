-- +goose Up
-- Preset IPSI/PERSILAT mendapat dua teknik kombinasi serangan (urutan prioritas seri regulasi:
-- 1+3, 3, 1+2, 2, 1+1, 1): "Kombinasi Pukulan + Tendangan" (1+2 = 3) dan "Kombinasi 2 Pukulan" (1+1 = 2).
-- Hanya preset bawaan yang sudah punya daftar teknik tetapi belum memuat kombinasi; ruleset buatan
-- pengguna tidak disentuh. Urutan prioritas seri turunan (nilai tertinggi lebih dulu, urutan daftar bila
-- sama) sudah menghasilkan 1+3, 3, 1+2, 2, 1+1, 1.
UPDATE rulesets
SET scoring = jsonb_set(
    scoring,
    '{techniques}',
    (scoring->'techniques') || '[
      {"type": "combo_1_2", "label": "Kombinasi Pukulan + Tendangan", "points": 3},
      {"type": "combo_1_1", "label": "Kombinasi 2 Pukulan", "points": 2}
    ]'::jsonb
)
WHERE name IN ('IPSI 2024', 'PERSILAT International')
  AND scoring ? 'techniques'
  AND NOT (scoring->'techniques' @> '[{"type": "combo_1_2"}]'::jsonb);

-- +goose Down
UPDATE rulesets
SET scoring = jsonb_set(
    scoring,
    '{techniques}',
    COALESCE((
      SELECT jsonb_agg(t)
      FROM jsonb_array_elements(scoring->'techniques') AS t
      WHERE t->>'type' NOT IN ('combo_1_2', 'combo_1_1')
    ), '[]'::jsonb)
)
WHERE name IN ('IPSI 2024', 'PERSILAT International') AND scoring ? 'techniques';
