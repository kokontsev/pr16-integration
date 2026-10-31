package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"example.com/pr16/internal/core"
	"example.com/pr16/internal/service"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Handler struct{ Service *service.Service }

func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != 204 {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func problem(w http.ResponseWriter, e error) {
	status, msg := 500, "internal_error"
	if errors.Is(e, core.ErrNotFound) {
		status, msg = 404, "not_found"
	}
	if errors.Is(e, core.ErrValidation) {
		status, msg = 400, "invalid_input"
	}
	Write(w, status, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		problem(w, core.ErrValidation)
		return false
	}
	return true
}
func id(w http.ResponseWriter, r *http.Request) (int64, bool) {
	v, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || v <= 0 {
		problem(w, core.ErrValidation)
		return 0, false
	}
	return v, true
}
func ctx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 5*time.Second)
}
func (h *Handler) Router() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("POST /api/v1/notes", h.Create)
	m.HandleFunc("GET /api/v1/notes", h.List)
	m.HandleFunc("GET /api/v1/notes/{id}", h.Get)
	m.HandleFunc("PATCH /api/v1/notes/{id}", h.Patch)
	m.HandleFunc("DELETE /api/v1/notes/{id}", h.Delete)
	return m
}

// Create godoc
// @Summary Create note
// @Tags notes
// @Accept json
// @Produce json
// @Param input body core.Create true "Note"
// @Success 201 {object} core.Note
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /notes [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in core.Create
	if !decode(w, r, &in) {
		return
	}
	c, cancel := ctx(r)
	defer cancel()
	v, e := h.Service.Create(c, in)
	if e != nil {
		problem(w, e)
		return
	}
	w.Header().Set("Location", "/api/v1/notes/"+strconv.FormatInt(v.ID, 10))
	Write(w, 201, v)
}

// Get godoc
// @Summary Get note
// @Tags notes
// @Produce json
// @Param id path int true "ID"
// @Success 200 {object} core.Note
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /notes/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	v, ok := id(w, r)
	if !ok {
		return
	}
	c, cancel := ctx(r)
	defer cancel()
	out, e := h.Service.Get(c, v)
	if e != nil {
		problem(w, e)
		return
	}
	Write(w, 200, out)
}

// List godoc
// @Summary List notes
// @Tags notes
// @Produce json
// @Param q query string false "Title search"
// @Param page query int false "Page" minimum(1) default(1)
// @Param limit query int false "Page size" minimum(1) maximum(100) default(20)
// @Success 200 {array} core.Note
// @Failure 400 {object} map[string]string
// @Router /notes [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, limit := 1, 20
	for k, p := range map[string]*int{"page": &page, "limit": &limit} {
		if raw := r.URL.Query().Get(k); raw != "" {
			v, e := strconv.Atoi(raw)
			if e != nil || v < 1 || v > 1000000 {
				problem(w, core.ErrValidation)
				return
			}
			*p = v
		}
	}
	if limit > 100 {
		problem(w, core.ErrValidation)
		return
	}
	c, cancel := ctx(r)
	defer cancel()
	out, e := h.Service.List(c, r.URL.Query().Get("q"), limit, (page-1)*limit)
	if e != nil {
		problem(w, e)
		return
	}
	Write(w, 200, out)
}

// Patch godoc
// @Summary Patch note
// @Tags notes
// @Accept json
// @Produce json
// @Param id path int true "ID"
// @Param input body core.Patch true "Changed fields"
// @Success 200 {object} core.Note
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /notes/{id} [patch]
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	v, ok := id(w, r)
	if !ok {
		return
	}
	var in core.Patch
	if !decode(w, r, &in) {
		return
	}
	c, cancel := ctx(r)
	defer cancel()
	out, e := h.Service.Patch(c, v, in)
	if e != nil {
		problem(w, e)
		return
	}
	Write(w, 200, out)
}

// Delete godoc
// @Summary Delete note
// @Tags notes
// @Param id path int true "ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /notes/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	v, ok := id(w, r)
	if !ok {
		return
	}
	c, cancel := ctx(r)
	defer cancel()
	if e := h.Service.Delete(c, v); e != nil {
		problem(w, e)
		return
	}
	Write(w, 204, nil)
}
