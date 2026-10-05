package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/herros27/digital-scoring/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

// SeedSummary records counts and key IDs created during database seeding.
type SeedSummary struct {
	RulesetIPSIID string
	CompetitionID string
	ArenaAID      string
	ArenaBID      string
	Officials     int
	Contingents   int
	Athletes      int
	Participants  int
	Matches       int
	JudgePins     int
}

// SeedData populates all database tables with comprehensive, realistic test data.
// ponytail: idempotent upserts allow running seed repeatedly without duplicate key violations.
func SeedData(db *sql.DB) (*SeedSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	summary := &SeedSummary{}

	// 1. Rulesets (IPSI 2024 & PERSILAT 7.0)
	scoringIPSI := map[string]any{
		"technique_points":  1,
		"knockdown_points":  2,
		"fall_points":       3,
		"max_penalties":     3,
		"penalty_threshold": 1,
		"round_duration":    120,
		"break_duration":    60,
		"total_rounds":      3,
	}
	rawScoringIPSI, _ := json.Marshal(scoringIPSI)

	var rulesetIPSIID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO rulesets (name, version, scoring, status)
		VALUES ('IPSI 2024', '2024.1', $1, 'active')
		ON CONFLICT (name, version) DO UPDATE SET scoring = EXCLUDED.scoring
		RETURNING id`, rawScoringIPSI).Scan(&rulesetIPSIID)
	if err != nil {
		return nil, fmt.Errorf("failed to seed IPSI ruleset: %w", err)
	}
	summary.RulesetIPSIID = rulesetIPSIID

	scoringPersilat := map[string]any{
		"technique_points":  1,
		"knockdown_points":  2,
		"fall_points":       3,
		"max_penalties":     3,
		"penalty_threshold": 1,
		"round_duration":    120,
		"break_duration":    60,
		"total_rounds":      3,
	}
	rawScoringPersilat, _ := json.Marshal(scoringPersilat)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO rulesets (name, version, scoring, status)
		VALUES ('PERSILAT International', '7.0', $1, 'active')
		ON CONFLICT (name, version) DO UPDATE SET scoring = EXCLUDED.scoring`, rawScoringPersilat)
	if err != nil {
		return nil, fmt.Errorf("failed to seed PERSILAT ruleset: %w", err)
	}

	// 2. Officials (Admin, Dewan, Wasit, Operator)
	officials := []struct {
		Name     string
		Email    string
		Role     string
		Password string
	}{
		{Name: "Administrator Utama", Email: "admin@silat.id", Role: "admin", Password: "DemoPass_2026"},
		{Name: "Ketua Dewan Juri", Email: "dewan@silat.id", Role: "dewan", Password: "DemoPass_2026"},
		{Name: "Wasit Utama", Email: "wasit@silat.id", Role: "wasit", Password: "DemoPass_2026"},
		{Name: "Operator Gelanggang", Email: "operator@silat.id", Role: "operator", Password: "DemoPass_2026"},
	}

	for _, off := range officials {
		hash, err := bcrypt.GenerateFromPassword([]byte(off.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash official password: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO officials (name, email, role, password_hash)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, role = EXCLUDED.role, password_hash = EXCLUDED.password_hash`,
			off.Name, off.Email, off.Role, string(hash))
		if err != nil {
			return nil, fmt.Errorf("failed to seed official %s: %w", off.Email, err)
		}
		summary.Officials++
	}

	// 3. Contingents
	contingentSeed := []struct {
		Name     string
		PIC      string
		Email    string
		Phone    string
		Password string
	}{
		{Name: "Tapak Suci Putera Muhammadiyah", PIC: "Ahmad Dahlan", Email: "tapaksuci@contingent.id", Phone: "081234567801", Password: "DemoPass_2026"},
		{Name: "Persaudaraan Setia Hati Terate (PSHT)", PIC: "Bambang Haryo", Email: "psht@contingent.id", Phone: "081234567802", Password: "DemoPass_2026"},
		{Name: "PPS Betako Merpati Putih", PIC: "Cecep Rahman", Email: "merpatiputih@contingent.id", Phone: "081234567803", Password: "DemoPass_2026"},
		{Name: "Kel. Silat Nasional Perisai Diri", PIC: "Danang Wijaya", Email: "perisaidiri@contingent.id", Phone: "081234567804", Password: "DemoPass_2026"},
	}

	contingentIDs := make(map[string]string)
	for _, c := range contingentSeed {
		hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash contingent password: %w", err)
		}
		var id string
		err = tx.QueryRowContext(ctx, `
			INSERT INTO contingents (name, pic_name, email, phone, password_hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, pic_name = EXCLUDED.pic_name, phone = EXCLUDED.phone, password_hash = EXCLUDED.password_hash
			RETURNING id`,
			c.Name, c.PIC, c.Email, c.Phone, string(hash)).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to seed contingent %s: %w", c.Email, err)
		}
		contingentIDs[c.Email] = id
		summary.Contingents++
	}

	// 4. Competition
	compName := "Kejuaraan Nasional Pencak Silat Piala Menpora 2026"
	var compID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM competitions WHERE name = $1 LIMIT 1`, compName).Scan(&compID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO competitions (name, location, start_date, end_date, ruleset_name, ruleset_id, status)
			VALUES ($1, 'GOR POPKI Cibubur, Jakarta Timur', '2026-10-10', '2026-10-15', 'ipsi', $2, 'active')
			RETURNING id`, compName, rulesetIPSIID).Scan(&compID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert competition: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query competition: %w", err)
	} else {
		_, _ = tx.ExecContext(ctx, `UPDATE competitions SET ruleset_id = $1, status = 'active' WHERE id = $2`, rulesetIPSIID, compID)
	}
	summary.CompetitionID = compID

	// 5. Arenas (Gelanggang A & Gelanggang B)
	arenaNames := []string{"Gelanggang A", "Gelanggang B"}
	arenaIDs := make(map[string]string)
	for _, aName := range arenaNames {
		var aID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM arenas WHERE competition_id = $1 AND name = $2 LIMIT 1`, compID, aName).Scan(&aID)
		if err == sql.ErrNoRows {
			err = tx.QueryRowContext(ctx, `
				INSERT INTO arenas (competition_id, name)
				VALUES ($1, $2)
				RETURNING id`, compID, aName).Scan(&aID)
			if err != nil {
				return nil, fmt.Errorf("failed to insert arena %s: %w", aName, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to query arena %s: %w", aName, err)
		}
		arenaIDs[aName] = aID
	}
	summary.ArenaAID = arenaIDs["Gelanggang A"]
	summary.ArenaBID = arenaIDs["Gelanggang B"]

	// 6. Categories
	categories := []struct {
		Name        string
		Gender      string
		WeightClass string
	}{
		{Name: "Tanding Dewasa Putra - Kelas A (45-50 Kg)", Gender: "male", WeightClass: "A"},
		{Name: "Tanding Dewasa Putra - Kelas B (50-55 Kg)", Gender: "male", WeightClass: "B"},
		{Name: "Tanding Dewasa Putra - Kelas C (55-60 Kg)", Gender: "male", WeightClass: "C"},
		{Name: "Tanding Dewasa Putra - Kelas D (60-65 Kg)", Gender: "male", WeightClass: "D"},
		{Name: "Tanding Dewasa Putra - Kelas E (65-70 Kg)", Gender: "male", WeightClass: "E"},
		{Name: "Tanding Dewasa Putra - Kelas F (70-75 Kg)", Gender: "male", WeightClass: "F"},
		{Name: "Tanding Dewasa Putra - Kelas G (75-80 Kg)", Gender: "male", WeightClass: "G"},
		{Name: "Tanding Dewasa Putra - Kelas H (80-85 Kg)", Gender: "male", WeightClass: "H"},
		{Name: "Tanding Dewasa Putra - Kelas I (85-90 Kg)", Gender: "male", WeightClass: "I"},
		{Name: "Tanding Dewasa Putra - Kelas J (90-95 Kg)", Gender: "male", WeightClass: "J"},
		{Name: "Tanding Dewasa Putri - Kelas A (45-50 Kg)", Gender: "female", WeightClass: "A"},
		{Name: "Tanding Dewasa Putri - Kelas B (50-55 Kg)", Gender: "female", WeightClass: "B"},
		{Name: "Tanding Dewasa Putri - Kelas C (55-60 Kg)", Gender: "female", WeightClass: "C"},
		{Name: "Tanding Dewasa Putri - Kelas D (60-65 Kg)", Gender: "female", WeightClass: "D"},
		{Name: "Tanding Dewasa Putri - Kelas E (65-70 Kg)", Gender: "female", WeightClass: "E"},
		{Name: "Tanding Dewasa Putri - Kelas F (70-75 Kg)", Gender: "female", WeightClass: "F"},
	}
	for _, cat := range categories {
		var catID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM categories WHERE competition_id = $1 AND name = $2 LIMIT 1`, compID, cat.Name).Scan(&catID)
		if err == sql.ErrNoRows {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO categories (competition_id, name, gender, weight_class, ruleset_id)
				VALUES ($1, $2, $3, $4, $5)`,
				compID, cat.Name, cat.Gender, cat.WeightClass, rulesetIPSIID)
			if err != nil {
				return nil, fmt.Errorf("failed to insert category %s: %w", cat.Name, err)
			}
		}
	}

	// 7. Athletes
	athletesSeed := []struct {
		Name           string
		Gender         string
		ContingentMail string
	}{
		{Name: "Eko Prasetyo", Gender: "male", ContingentMail: "tapaksuci@contingent.id"},
		{Name: "Fajar Nugroho", Gender: "male", ContingentMail: "psht@contingent.id"},
		{Name: "Gilang Ramadhan", Gender: "male", ContingentMail: "merpatiputih@contingent.id"},
		{Name: "Hendra Setiawan", Gender: "male", ContingentMail: "perisaidiri@contingent.id"},
		{Name: "Ilham Maulana", Gender: "male", ContingentMail: "tapaksuci@contingent.id"},
		{Name: "Joko Susilo", Gender: "male", ContingentMail: "psht@contingent.id"},
		{Name: "Siti Rahmawati", Gender: "female", ContingentMail: "merpatiputih@contingent.id"},
		{Name: "Dewi Sartika", Gender: "female", ContingentMail: "perisaidiri@contingent.id"},
	}

	athleteIDs := make(map[string]string)
	for _, ath := range athletesSeed {
		cID := contingentIDs[ath.ContingentMail]
		var aID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM athletes WHERE name = $1 AND contingent_id = $2 LIMIT 1`, ath.Name, cID).Scan(&aID)
		if err == sql.ErrNoRows {
			err = tx.QueryRowContext(ctx, `
				INSERT INTO athletes (name, gender, team, contingent_id)
				VALUES ($1, $2, (SELECT name FROM contingents WHERE id = $3), $3)
				RETURNING id`, ath.Name, ath.Gender, cID).Scan(&aID)
			if err != nil {
				return nil, fmt.Errorf("failed to insert athlete %s: %w", ath.Name, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to query athlete %s: %w", ath.Name, err)
		}
		athleteIDs[ath.Name] = aID
		summary.Athletes++
	}

	// 8. Participants (Registered in competition for Kelas A Putra)
	participantsSeed := []struct {
		AthleteName    string
		SeedNumber     int
		ContingentMail string
	}{
		{AthleteName: "Eko Prasetyo", SeedNumber: 1, ContingentMail: "tapaksuci@contingent.id"},
		{AthleteName: "Fajar Nugroho", SeedNumber: 2, ContingentMail: "psht@contingent.id"},
		{AthleteName: "Gilang Ramadhan", SeedNumber: 3, ContingentMail: "merpatiputih@contingent.id"},
		{AthleteName: "Hendra Setiawan", SeedNumber: 4, ContingentMail: "perisaidiri@contingent.id"},
	}

	participantIDs := make(map[string]string)
	for _, p := range participantsSeed {
		athID := athleteIDs[p.AthleteName]
		cID := contingentIDs[p.ContingentMail]
		var partID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM participants WHERE competition_id = $1 AND athlete_id = $2 LIMIT 1`, compID, athID).Scan(&partID)
		if err == sql.ErrNoRows {
			err = tx.QueryRowContext(ctx, `
				INSERT INTO participants (competition_id, name, weight_class, gender, athlete_id, seed, registration_status, contingent_id)
				VALUES ($1, $2, 'A', 'male', $3, $4, 'verified', $5)
				RETURNING id`, compID, p.AthleteName, athID, p.SeedNumber, cID).Scan(&partID)
			if err != nil {
				return nil, fmt.Errorf("failed to insert participant %s: %w", p.AthleteName, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to query participant %s: %w", p.AthleteName, err)
		}
		participantIDs[p.AthleteName] = partID
		summary.Participants++
	}

	// 9. Bracket Slots (Semifinal 1, Semifinal 2, Final)
	// Final slot
	var finalSlotID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM bracket_slots
		WHERE competition_id = $1 AND weight_class = 'A' AND gender = 'male' AND round = 2 AND slot_index = 1 LIMIT 1`,
		compID).Scan(&finalSlotID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO bracket_slots (competition_id, weight_class, gender, round, slot_index, is_bye, next_slot_id)
			VALUES ($1, 'A', 'male', 2, 1, false, NULL)
			RETURNING id`, compID).Scan(&finalSlotID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert final bracket slot: %w", err)
		}
	}

	// Semifinal 1 (Eko Prasetyo)
	var semi1SlotID string
	p1ID := participantIDs["Eko Prasetyo"]
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM bracket_slots
		WHERE competition_id = $1 AND weight_class = 'A' AND gender = 'male' AND round = 1 AND slot_index = 1 LIMIT 1`,
		compID).Scan(&semi1SlotID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO bracket_slots (competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, next_slot_id)
			VALUES ($1, 'A', 'male', 1, 1, $2, false, $3)
			RETURNING id`, compID, p1ID, finalSlotID).Scan(&semi1SlotID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert semi 1 bracket slot: %w", err)
		}
	}

	// Semifinal 2 (Gilang Ramadhan)
	var semi2SlotID string
	p3ID := participantIDs["Gilang Ramadhan"]
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM bracket_slots
		WHERE competition_id = $1 AND weight_class = 'A' AND gender = 'male' AND round = 1 AND slot_index = 2 LIMIT 1`,
		compID).Scan(&semi2SlotID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO bracket_slots (competition_id, weight_class, gender, round, slot_index, participant_id, is_bye, next_slot_id)
			VALUES ($1, 'A', 'male', 1, 2, $2, false, $3)
			RETURNING id`, compID, p3ID, finalSlotID).Scan(&semi2SlotID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert semi 2 bracket slot: %w", err)
		}
	}

	// 10. Matches
	// Match 1 in Gelanggang A: Eko Prasetyo (Red) vs Fajar Nugroho (Blue)
	p2ID := participantIDs["Fajar Nugroho"]
	var match1ID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM matches WHERE competition_id = $1 AND arena_id = $2 AND bracket_slot_id = $3 LIMIT 1`,
		compID, summary.ArenaAID, semi1SlotID).Scan(&match1ID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO matches (competition_id, arena_id, bracket_slot_id, athlete_red_id, athlete_blue_id, current_round, status)
			VALUES ($1, $2, $3, $4, $5, 1, 'idle')
			RETURNING id`, compID, summary.ArenaAID, semi1SlotID, p1ID, p2ID).Scan(&match1ID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert match 1: %w", err)
		}
		summary.Matches++
	}

	// Match 2 in Gelanggang B: Gilang Ramadhan (Red) vs Hendra Setiawan (Blue)
	p4ID := participantIDs["Hendra Setiawan"]
	var match2ID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM matches WHERE competition_id = $1 AND arena_id = $2 AND bracket_slot_id = $3 LIMIT 1`,
		compID, summary.ArenaBID, semi2SlotID).Scan(&match2ID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO matches (competition_id, arena_id, bracket_slot_id, athlete_red_id, athlete_blue_id, current_round, status)
			VALUES ($1, $2, $3, $4, $5, 1, 'idle')
			RETURNING id`, compID, summary.ArenaBID, semi2SlotID, p3ID, p4ID).Scan(&match2ID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert match 2: %w", err)
		}
		summary.Matches++
	}

	// 11. Judge & Referee PINs (Valid for 365 days)
	// Gelanggang A PINs: Wasit=111111, Juri 1=123001, Juri 2=123002, Juri 3=123003
	// Gelanggang B PINs: Wasit=222222, Juri 1=234001, Juri 2=234002, Juri 3=234003
	pinsToSeed := []struct {
		ArenaID    string
		Role       string
		SeatNumber int
		PIN        string
	}{
		{ArenaID: summary.ArenaAID, Role: "wasit", SeatNumber: 0, PIN: "111111"},
		{ArenaID: summary.ArenaAID, Role: "juri", SeatNumber: 1, PIN: "123001"},
		{ArenaID: summary.ArenaAID, Role: "juri", SeatNumber: 2, PIN: "123002"},
		{ArenaID: summary.ArenaAID, Role: "juri", SeatNumber: 3, PIN: "123003"},

		{ArenaID: summary.ArenaBID, Role: "wasit", SeatNumber: 0, PIN: "222222"},
		{ArenaID: summary.ArenaBID, Role: "juri", SeatNumber: 1, PIN: "234001"},
		{ArenaID: summary.ArenaBID, Role: "juri", SeatNumber: 2, PIN: "234002"},
		{ArenaID: summary.ArenaBID, Role: "juri", SeatNumber: 3, PIN: "234003"},
	}

	expiry := time.Now().AddDate(1, 0, 0)
	for _, p := range pinsToSeed {
		hash, err := auth.HashPIN(p.PIN)
		if err != nil {
			return nil, fmt.Errorf("failed to hash pin: %w", err)
		}

		var existingID string
		err = tx.QueryRowContext(ctx, `
			SELECT id FROM judge_pins
			WHERE competition_id = $1 AND arena_id = $2 AND role = $3 AND COALESCE(seat_number, 0) = $4 LIMIT 1`,
			compID, p.ArenaID, p.Role, p.SeatNumber).Scan(&existingID)
		if err == sql.ErrNoRows {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO judge_pins (competition_id, arena_id, role, seat_number, raw_pin, pin_hash, expires_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				compID, p.ArenaID, p.Role, p.SeatNumber, p.PIN, hash, expiry)
			if err != nil {
				return nil, fmt.Errorf("failed to insert judge pin for arena %s %s:%d: %w", p.ArenaID, p.Role, p.SeatNumber, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to check existing pin: %w", err)
		} else {
			_, _ = tx.ExecContext(ctx, `
				UPDATE judge_pins SET raw_pin = $1, pin_hash = $2, expires_at = $3 WHERE id = $4`,
				p.PIN, hash, expiry, existingID)
		}
		summary.JudgePins++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return summary, nil
}
