package store

import (
	"database/sql"
	"time"
)

type OfficialStore struct {
	DB *sql.DB
}

func (s *OfficialStore) Create(o *Official, passwordHash string) error {
	query := `
		INSERT INTO officials (name, email, role, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	o.CreatedAt = time.Now()
	return s.DB.QueryRow(query, o.Name, o.Email, o.Role, passwordHash, o.CreatedAt).Scan(&o.ID, &o.CreatedAt)
}

func (s *OfficialStore) GetByID(id string) (*Official, error) {
	o := &Official{}
	query := `SELECT id, name, email, role, created_at FROM officials WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(&o.ID, &o.Name, &o.Email, &o.Role, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *OfficialStore) GetByEmail(email string) (*Official, string, error) {
	o := &Official{}
	var passwordHash string
	query := `SELECT id, name, email, role, password_hash, created_at FROM officials WHERE email = $1`
	err := s.DB.QueryRow(query, email).Scan(&o.ID, &o.Name, &o.Email, &o.Role, &passwordHash, &o.CreatedAt)
	if err != nil {
		return nil, "", err
	}
	return o, passwordHash, nil
}

func (s *OfficialStore) List() ([]Official, error) {
	query := `SELECT id, name, email, role, created_at FROM officials ORDER BY created_at DESC`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Official
	for rows.Next() {
		var o Official
		if err := rows.Scan(&o.ID, &o.Name, &o.Email, &o.Role, &o.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	if list == nil {
		list = []Official{}
	}
	return list, rows.Err()
}

func (s *OfficialStore) Update(o *Official) error {
	query := `UPDATE officials SET name = $1, role = $2 WHERE id = $3`
	_, err := s.DB.Exec(query, o.Name, o.Role, o.ID)
	return err
}

func (s *OfficialStore) Delete(id string) error {
	query := `DELETE FROM officials WHERE id = $1`
	_, err := s.DB.Exec(query, id)
	return err
}

// UpdatePassword mengganti hash password akun.
func (s *OfficialStore) UpdatePassword(id, passwordHash string) error {
	_, err := s.DB.Exec(`UPDATE officials SET password_hash = $1 WHERE id = $2`, passwordHash, id)
	return err
}

// CountByRole jumlah akun dengan peran tertentu (dipakai agar admin terakhir tidak terhapus/diturunkan).
func (s *OfficialStore) CountByRole(role string) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM officials WHERE role = $1`, role).Scan(&n)
	return n, err
}
