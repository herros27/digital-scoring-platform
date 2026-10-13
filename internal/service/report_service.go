package service

import (
	"sort"

	"github.com/herros27/digital-scoring/internal/domain"
	"github.com/herros27/digital-scoring/internal/store"
)

type MatchReportRepository interface {
	ListByCompetition(competitionID string) ([]store.Match, error)
	// MedalPlacements medali per tim dari hasil bagan (juara, finalis, kalah semifinal).
	MedalPlacements(competitionID string) ([]store.MedalPlacement, error)
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

// GetMedalStandings klasemen medali per tim dari hasil bagan: satu emas (juara), satu perak (finalis), dan perunggu
// untuk tiap yang kalah di semifinal pada setiap kategori. Kemenangan di babak awal tidak memberi medali.
func (s *ReportService) GetMedalStandings(competitionID string) ([]domain.MedalStanding, error) {
	if competitionID == "" {
		return nil, domain.ErrInvalidInput
	}
	placements, err := s.matchRepo.MedalPlacements(competitionID)
	if err != nil {
		return nil, err
	}

	byTeam := make(map[string]*domain.MedalStanding)
	for _, p := range placements {
		t, ok := byTeam[p.Team]
		if !ok {
			t = &domain.MedalStanding{Team: p.Team}
			byTeam[p.Team] = t
		}
		switch p.Medal {
		case "gold":
			t.Gold++
		case "silver":
			t.Silver++
		case "bronze":
			t.Bronze++
		default:
			continue
		}
		t.Total++
	}

	standings := make([]domain.MedalStanding, 0, len(byTeam))
	for _, t := range byTeam {
		standings = append(standings, *t)
	}
	sort.Slice(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]
		if a.Gold != b.Gold {
			return a.Gold > b.Gold
		}
		if a.Silver != b.Silver {
			return a.Silver > b.Silver
		}
		if a.Bronze != b.Bronze {
			return a.Bronze > b.Bronze
		}
		return a.Team < b.Team
	})
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
