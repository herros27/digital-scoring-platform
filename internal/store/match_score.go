package store

import "database/sql"

type MatchScoreStore struct {
	DB *sql.DB
}

func (s *MatchScoreStore) Upsert(sc *MatchScore) error {
	query := `
		INSERT INTO match_scores (match_id, round, athlete_red_score, athlete_blue_score, red_penalties, blue_penalties)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (match_id, round)
		DO UPDATE SET
			athlete_red_score = $3,
			athlete_blue_score = $4,
			red_penalties = $5,
			blue_penalties = $6,
			updated_at = NOW()`
	_, err := s.DB.Exec(query,
		sc.MatchID, sc.Round,
		sc.AthleteRedScore, sc.AthleteBlueScore,
		sc.RedPenalties, sc.BluePenalties,
	)
	return err
}

func (s *MatchScoreStore) GetByMatch(matchID string) ([]MatchScore, error) {
	rows, err := s.DB.Query(`SELECT id, match_id, round, athlete_red_score, athlete_blue_score, red_penalties, blue_penalties, updated_at FROM match_scores WHERE match_id = $1 ORDER BY round`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MatchScore
	for rows.Next() {
		var sc MatchScore
		rows.Scan(&sc.ID, &sc.MatchID, &sc.Round, &sc.AthleteRedScore, &sc.AthleteBlueScore, &sc.RedPenalties, &sc.BluePenalties, &sc.UpdatedAt)
		list = append(list, sc)
	}
	return list, rows.Err()
}
