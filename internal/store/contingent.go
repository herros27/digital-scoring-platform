package store

import (
	"database/sql"
	"time"
)

type ContingentStore struct {
	DB *sql.DB
}

func (s *ContingentStore) Create(c *Contingent, passwordHash string) error {
	query := `
		INSERT INTO contingents (name, pic_name, email, phone, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	c.CreatedAt = time.Now()
	return s.DB.QueryRow(query, c.Name, c.PicName, c.Email, c.Phone, passwordHash, c.CreatedAt).Scan(&c.ID, &c.CreatedAt)
}

func (s *ContingentStore) GetByID(id string) (*Contingent, error) {
	c := &Contingent{}
	query := `SELECT id, name, pic_name, email, phone, created_at FROM contingents WHERE id = $1`
	err := s.DB.QueryRow(query, id).Scan(&c.ID, &c.Name, &c.PicName, &c.Email, &c.Phone, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ContingentStore) GetByEmail(email string) (*Contingent, string, error) {
	c := &Contingent{}
	var passwordHash string
	query := `SELECT id, name, pic_name, email, phone, password_hash, created_at FROM contingents WHERE email = $1`
	err := s.DB.QueryRow(query, email).Scan(&c.ID, &c.Name, &c.PicName, &c.Email, &c.Phone, &passwordHash, &c.CreatedAt)
	if err != nil {
		return nil, "", err
	}
	return c, passwordHash, nil
}

func (s *ContingentStore) List() ([]Contingent, error) {
	query := `SELECT id, name, pic_name, email, phone, created_at FROM contingents ORDER BY name ASC`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Contingent
	for rows.Next() {
		var c Contingent
		if err := rows.Scan(&c.ID, &c.Name, &c.PicName, &c.Email, &c.Phone, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if list == nil {
		list = []Contingent{}
	}
	return list, rows.Err()
}

// UpdatePassword mengganti hash password akun kontingen.
func (s *ContingentStore) UpdatePassword(id, passwordHash string) error {
	res, err := s.DB.Exec(`UPDATE contingents SET password_hash = $1 WHERE id = $2`, passwordHash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
