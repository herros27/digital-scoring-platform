package domain

import "errors"

var (
	ErrNotFound            = errors.New("resource not found")
	ErrCompetitionNotFound = errors.New("competition not found")
	ErrMatchNotFound       = errors.New("match not found")
	ErrArenaNotFound       = errors.New("arena not found")
	ErrAlreadyActive       = errors.New("competition is already active")
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrConflict            = errors.New("resource conflict")
)
