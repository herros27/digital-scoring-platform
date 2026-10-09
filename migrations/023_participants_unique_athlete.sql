-- +goose Up
-- Satu atlet hanya boleh terdaftar satu kali pada satu kejuaraan. Sebelumnya POST /athletes dengan
-- athlete_id yang sama menghasilkan baris peserta ganda.

-- Bersihkan duplikat lama: pertahankan pendaftaran paling awal, buang salinan yang belum dipakai
-- bagan maupun pertandingan (salinan yang sudah dirujuk tidak disentuh agar riwayat tetap utuh).
DELETE FROM participants p
USING participants k
WHERE p.athlete_id IS NOT NULL
  AND p.competition_id = k.competition_id
  AND p.athlete_id = k.athlete_id
  AND (k.created_at, k.id::text) < (p.created_at, p.id::text)
  AND NOT EXISTS (SELECT 1 FROM bracket_slots b WHERE b.participant_id = p.id OR b.winner_id = p.id)
  AND NOT EXISTS (SELECT 1 FROM matches m
                  WHERE m.athlete_red_id = p.id OR m.athlete_blue_id = p.id OR m.winner_id = p.id);

-- Indeks unik dibuat hanya bila tidak tersisa duplikat yang terkunci referensi; selain itu migrasi
-- tidak boleh menggagalkan startup, jadi indeks dilewati (dan validasi aplikasi + 409 tetap berlaku
-- untuk data baru setelah duplikat itu dibereskan manual).
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM participants WHERE athlete_id IS NOT NULL
    GROUP BY competition_id, athlete_id HAVING COUNT(*) > 1
  ) THEN
    RAISE NOTICE 'ux_participants_competition_athlete dilewati: masih ada peserta ganda yang dirujuk bagan/pertandingan';
  ELSE
    CREATE UNIQUE INDEX IF NOT EXISTS ux_participants_competition_athlete
      ON participants (competition_id, athlete_id) WHERE athlete_id IS NOT NULL;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS ux_participants_competition_athlete;
