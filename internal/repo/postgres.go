package repo

import (
	"context"
	"database/sql"
	"errors"
	"example.com/pr16/internal/core"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type Postgres struct{ DB *sql.DB }

func Open(c context.Context, dsn string, max int) (*Postgres, error) {
	d, e := sql.Open("pgx", dsn)
	if e != nil {
		return nil, e
	}
	d.SetMaxOpenConns(max)
	d.SetMaxIdleConns(max / 2)
	d.SetConnMaxLifetime(30 * time.Minute)
	d.SetConnMaxIdleTime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()
	if e = d.PingContext(ctx); e != nil {
		d.Close()
		return nil, e
	}
	_, e = d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS notes(id BIGSERIAL PRIMARY KEY,title TEXT NOT NULL,content TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now(),updated_at TIMESTAMPTZ NOT NULL DEFAULT now());`)
	if e != nil {
		d.Close()
		return nil, e
	}
	return &Postgres{d}, nil
}
func scan(row interface{ Scan(...any) error }) (core.Note, error) {
	var n core.Note
	e := row.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		e = core.ErrNotFound
	}
	return n, e
}

const columns = `id,title,content,created_at,updated_at`

func (p *Postgres) Create(c context.Context, in core.Create) (core.Note, error) {
	return scan(p.DB.QueryRowContext(c, `INSERT INTO notes(title,content)VALUES($1,$2)RETURNING `+columns, in.Title, in.Content))
}
func (p *Postgres) Get(c context.Context, id int64) (core.Note, error) {
	return scan(p.DB.QueryRowContext(c, `SELECT `+columns+` FROM notes WHERE id=$1`, id))
}
func (p *Postgres) List(c context.Context, q string, limit, offset int) ([]core.Note, error) {
	rows, e := p.DB.QueryContext(c, `SELECT `+columns+` FROM notes WHERE strpos(lower(title),lower($1))>0 ORDER BY id DESC LIMIT $2 OFFSET $3`, q, limit, offset)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []core.Note{}
	for rows.Next() {
		n, e := scan(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
func (p *Postgres) Patch(c context.Context, id int64, in core.Patch) (core.Note, error) {
	return scan(p.DB.QueryRowContext(c, `UPDATE notes SET title=COALESCE($2,title),content=COALESCE($3,content),updated_at=clock_timestamp() WHERE id=$1 RETURNING `+columns, id, in.Title, in.Content))
}
func (p *Postgres) Delete(c context.Context, id int64) error {
	v, e := p.DB.ExecContext(c, `DELETE FROM notes WHERE id=$1`, id)
	if e != nil {
		return e
	}
	n, e := v.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return core.ErrNotFound
	}
	return nil
}
