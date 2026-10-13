-- +goose Up
-- Kategori lama (dibuat sebelum ada kolom min_weight/max_weight, mis. dari seed) hanya memuat rentang bobot
-- di namanya, mis. "Tanding Dewasa Putra - Kelas A (45-50 Kg)". Rentang itu dipindahkan ke min_weight/
-- max_weight supaya verifikasi timbang menegakkan batasnya (tanpa toleransi, batas inklusif seperti preset UI).
--
-- Hanya kategori yang KEDUA batasnya masih kosong dan namanya memuat tepat pola "(angka-angka Kg)" dengan
-- batas bawah <= batas atas. Kategori yang batasnya sudah diisi atau namanya tidak berpola tidak disentuh.
-- Aman dijalankan ulang.
UPDATE categories c
SET min_weight = x.lo,
    max_weight = x.hi
FROM (
  SELECT id,
         replace(m[1], ',', '.')::numeric AS lo,
         replace(m[2], ',', '.')::numeric AS hi
  FROM (
    SELECT id, regexp_match(name, '\((\d+(?:[.,]\d+)?)\s*-\s*(\d+(?:[.,]\d+)?)\s*[Kk][Gg]\)') AS m
    FROM categories
    WHERE min_weight IS NULL AND max_weight IS NULL
  ) parsed
  WHERE m IS NOT NULL
) x
WHERE c.id = x.id
  AND x.lo <= x.hi
  AND x.hi > 0
  AND x.hi < 1000;

-- +goose Down
-- Migrasi data satu arah: batas bobot yang sudah diisi tidak dikosongkan kembali.
SELECT 1;
