-- +goose Up
-- Memindahkan data lama (sebelum sistem kategori) ke sistem terbaru: peserta dan slot bagan yang
-- category_id-nya masih kosong dihubungkan ke kategori resmi kejuaraannya, supaya bagan lama tampil
-- di bawah kategori (bagan terisolasi per category_id) dan bisa di-generate ulang dari kategori itu.
--
-- Pemetaan memakai (kejuaraan, kelas bobot, gender). Gender dinormalkan (putra/male/m -> male,
-- putri/female/f -> female). Bila beberapa kategori cocok, dipilih yang golongan umurnya 'dewasa'
-- (data lama dibuat sebelum ada golongan umur; migrasi 022 memberi semua kategori lama 'dewasa').
-- Bila masih ambigu atau tidak ada kategori yang cocok, baris dibiarkan apa adanya (tetap bisa
-- diakses lewat jalur lama) agar tidak ada peserta yang salah tempat.
-- Aman dijalankan ulang: hanya menyentuh baris dengan category_id kosong.

DROP TABLE IF EXISTS legacy_category_map;
CREATE TEMP TABLE legacy_category_map ON COMMIT DROP AS
SELECT competition_id, weight_class, g, category_id
FROM (
  SELECT c.competition_id,
         c.weight_class,
         CASE WHEN lower(c.gender) IN ('male', 'putra', 'm', 'l') THEN 'male'
              WHEN lower(c.gender) IN ('female', 'putri', 'f', 'p') THEN 'female'
              ELSE lower(c.gender) END AS g,
         CASE WHEN count(*) = 1 THEN (array_agg(c.id))[1]
              WHEN count(*) FILTER (WHERE c.age_group = 'dewasa') = 1
                THEN (array_agg(c.id) FILTER (WHERE c.age_group = 'dewasa'))[1]
         END AS category_id
  FROM categories c
  GROUP BY c.competition_id, c.weight_class,
           CASE WHEN lower(c.gender) IN ('male', 'putra', 'm', 'l') THEN 'male'
                WHEN lower(c.gender) IN ('female', 'putri', 'f', 'p') THEN 'female'
                ELSE lower(c.gender) END
) m
WHERE category_id IS NOT NULL;

UPDATE participants p
SET category_id = m.category_id
FROM legacy_category_map m
WHERE p.category_id IS NULL
  AND m.competition_id = p.competition_id
  AND m.weight_class = p.weight_class
  AND m.g = CASE WHEN lower(p.gender) IN ('male', 'putra', 'm', 'l') THEN 'male'
                 WHEN lower(p.gender) IN ('female', 'putri', 'f', 'p') THEN 'female'
                 ELSE lower(p.gender) END;

UPDATE bracket_slots b
SET category_id = m.category_id
FROM legacy_category_map m
WHERE b.category_id IS NULL
  AND m.competition_id = b.competition_id
  AND m.weight_class = b.weight_class
  AND m.g = CASE WHEN lower(b.gender) IN ('male', 'putra', 'm', 'l') THEN 'male'
                 WHEN lower(b.gender) IN ('female', 'putri', 'f', 'p') THEN 'female'
                 ELSE lower(b.gender) END;

-- +goose Down
-- Migrasi data satu arah: tautan kategori yang valid tidak dicabut kembali.
SELECT 1;
