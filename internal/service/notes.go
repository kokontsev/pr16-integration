package service

import (
	"context"
	"example.com/pr16/internal/core"
	"strings"
	"unicode/utf8"
)

type Repository interface {
	Create(context.Context, core.Create) (core.Note, error)
	Get(context.Context, int64) (core.Note, error)
	List(context.Context, string, int, int) ([]core.Note, error)
	Patch(context.Context, int64, core.Patch) (core.Note, error)
	Delete(context.Context, int64) error
}
type Service struct{ repo Repository }

func New(r Repository) *Service { return &Service{r} }
func valid(s string) bool       { return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= 200 }
func (s *Service) Create(c context.Context, in core.Create) (core.Note, error) {
	in.Title = strings.TrimSpace(in.Title)
	if !valid(in.Title) {
		return core.Note{}, core.ErrValidation
	}
	return s.repo.Create(c, in)
}
func (s *Service) Get(c context.Context, id int64) (core.Note, error) { return s.repo.Get(c, id) }
func (s *Service) List(c context.Context, q string, limit, offset int) ([]core.Note, error) {
	return s.repo.List(c, q, limit, offset)
}
func (s *Service) Patch(c context.Context, id int64, in core.Patch) (core.Note, error) {
	if in.Title == nil && in.Content == nil {
		return core.Note{}, core.ErrValidation
	}
	if in.Title != nil {
		v := strings.TrimSpace(*in.Title)
		if !valid(v) {
			return core.Note{}, core.ErrValidation
		}
		in.Title = &v
	}
	return s.repo.Patch(c, id, in)
}
func (s *Service) Delete(c context.Context, id int64) error { return s.repo.Delete(c, id) }
