package domain

import "errors"

var (
	ErrNotFound               = errors.New("resource not found")
	ErrCompetitionNotFound    = errors.New("competition not found")
	ErrMatchNotFound          = errors.New("match not found")
	ErrArenaNotFound          = errors.New("arena not found")
	ErrAlreadyActive          = errors.New("competition is already active")
	ErrInvalidInput           = errors.New("invalid input")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
	ErrConflict               = errors.New("resource conflict")
	ErrArenaBusy              = errors.New("gelanggang sedang dipakai partai lain yang masih aktif; selesaikan atau batalkan partai itu terlebih dahulu")
	ErrInvalidStateTransition = errors.New("invalid state transition")
)
