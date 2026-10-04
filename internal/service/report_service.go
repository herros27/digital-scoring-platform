package service

import (
	"sort"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/store"
)

type MatchReportRepository interface {
	ListByCompetition(competitionID string) ([]store.Match, error)
}

type ReportService struct {
	compRepo     CompetitionRepository
	arenaRepo    ArenaRepository
	categoryRepo CategoryRepository
	athleteRepo  AthleteRepository
	matchRepo    MatchReportRepository
}

func NewReportService(
	compRepo CompetitionRepository,
	arenaRepo ArenaRepository,
	categoryRepo CategoryRepository,
	athleteRepo AthleteRepository,
	matchRepo MatchReportRepository,
) *ReportService {
	return &ReportService{
		compRepo:     compRepo,
		arenaRepo:    arenaRepo,
		categoryRepo: categoryRepo,
		athleteRepo:  athleteRepo,
		matchRepo:    matchRepo,
	}
}

func (s *ReportService) GetSummary(competitionID string) (*domain.CompetitionSummary, error) {
	if competitionID == "" {
		return nil, domain.ErrInvalidInput
	}
	comp, err := s.compRepo.GetByID(competitionID)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, domain.ErrCompetitionNotFound
	}

	arenas, _ := s.arenaRepo.ListByCompetition(competitionID)
	categories, _ := s.categoryRepo.ListByCompetition(competitionID)
	athletes, _ := s.athleteRepo.ListByCompetition(competitionID)
	matches, _ := s.matchRepo.ListByCompetition(competitionID)

	statusMap := make(map[string]int)
	for _, m := range matches {
		status := m.Status
		if status == "" {
			status = "scheduled"
		}
		statusMap[status]++
	}

	summary := &domain.CompetitionSummary{
		CompetitionID:   comp.ID,
		CompetitionName: comp.Name,
		TotalArenas:     len(arenas),
		TotalCategories: len(categories),
		TotalAthletes:   len(athletes),
		TotalMatches:    len(matches),
		MatchesByStatus: statusMap,
	}
	return summary, nil
}

func (s *ReportService) GetMedalStandings(competitionID string) ([]domain.MedalStanding, error) {
	if competitionID == "" {
		return nil, domain.ErrInvalidInput
	}
	matches, err := s.matchRepo.ListByCompetition(competitionID)
	if err != nil {
		return nil, err
	}

	athletes, _ := s.athleteRepo.List()
	teamByAthlete := make(map[string]string)
	for _, a := range athletes {
		team := a.Team
		if team == "" {
			team = "Independen"
		}
		teamByAthlete[a.ID] = team
	}

	teamMedals := make(map[string]*domain.MedalStanding)
	for _, m := range matches {
		if m.Status != "finished" || m.WinnerID == "" {
			continue
		}

		winnerTeam := teamByAthlete[m.WinnerID]
		if winnerTeam == "" {
			winnerTeam = "Umum"
		}

		if _, ok := teamMedals[winnerTeam]; !ok {
			teamMedals[winnerTeam] = &domain.MedalStanding{Team: winnerTeam}
		}

		// Winner gets a gold/win point
		teamMedals[winnerTeam].Gold++
		teamMedals[winnerTeam].Total++

		// Runner-up (loser) gets silver
		loserID := m.AthleteRedID
		if m.WinnerID == m.AthleteRedID {
			loserID = m.AthleteBlueID
		}
		if loserID != "" {
			loserTeam := teamByAthlete[loserID]
			if loserTeam == "" {
				loserTeam = "Umum"
			}
			if _, ok := teamMedals[loserTeam]; !ok {
				teamMedals[loserTeam] = &domain.MedalStanding{Team: loserTeam}
			}
			teamMedals[loserTeam].Silver++
			teamMedals[loserTeam].Total++
		}
	}

	var standings []domain.MedalStanding
	for _, standing := range teamMedals {
		standings = append(standings, *standing)
	}

	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Gold != standings[j].Gold {
			return standings[i].Gold > standings[j].Gold
		}
		if standings[i].Silver != standings[j].Silver {
			return standings[i].Silver > standings[j].Silver
		}
		return standings[i].Bronze > standings[j].Bronze
	})

	if standings == nil {
		standings = []domain.MedalStanding{}
	}

	return standings, nil
}

func (s *ReportService) ExportCompetition(competitionID string) (*domain.CompetitionExport, error) {
	summary, err := s.GetSummary(competitionID)
	if err != nil {
		return nil, err
	}
	comp, err := s.compRepo.GetByID(competitionID)
	if err != nil {
		return nil, err
	}
	medals, _ := s.GetMedalStandings(competitionID)
	categories, _ := s.categoryRepo.ListByCompetition(competitionID)
	matches, _ := s.matchRepo.ListByCompetition(competitionID)

	export := &domain.CompetitionExport{
		Competition: comp,
		Summary:     summary,
		Medals:      medals,
		Categories:  categories,
		Matches:     matches,
	}
	return export, nil
}
