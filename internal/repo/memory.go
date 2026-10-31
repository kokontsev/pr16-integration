package repo

import (
	"context"
	"example.com/pr16/internal/core"
	"sort"
	"strings"
	"sync"
	"time"
)

type Memory struct {
	mu    sync.RWMutex
	notes map[int64]core.Note
	next  int64
}

func NewMemory() *Memory { return &Memory{notes: map[int64]core.Note{}} }
func (m *Memory) Create(ctx context.Context, in core.Create) (core.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	now := time.Now().UTC()
	v := core.Note{ID: m.next, Title: in.Title, Content: in.Content, CreatedAt: now, UpdatedAt: now}
	m.notes[v.ID] = v
	return v, nil
}
func (m *Memory) Get(ctx context.Context, id int64) (core.Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.notes[id]
	if !ok {
		return v, core.ErrNotFound
	}
	return v, nil
}
func (m *Memory) List(ctx context.Context, q string, limit, offset int) ([]core.Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []core.Note{}
	for _, v := range m.notes {
		if strings.Contains(strings.ToLower(v.Title), strings.ToLower(q)) {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if offset >= len(out) {
		return []core.Note{}, nil
	}
	out = out[offset:]
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (m *Memory) Patch(ctx context.Context, id int64, in core.Patch) (core.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.notes[id]
	if !ok {
		return v, core.ErrNotFound
	}
	if in.Title != nil {
		v.Title = *in.Title
	}
	if in.Content != nil {
		v.Content = *in.Content
	}
	v.UpdatedAt = time.Now().UTC()
	m.notes[id] = v
	return v, nil
}
func (m *Memory) Delete(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.notes[id]; !ok {
		return core.ErrNotFound
	}
	delete(m.notes, id)
	return nil
}
