package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/yourusername/bf-offers/webclient/internal/models"
	"github.com/yourusername/bf-offers/webclient/internal/repository"
)

type ImportTemplateHandler struct {
	repo *repository.ImportTemplateRepository
}

func NewImportTemplateHandler(repo *repository.ImportTemplateRepository) *ImportTemplateHandler {
	return &ImportTemplateHandler{repo: repo}
}

// GetAllTemplates returns all import templates
func (h *ImportTemplateHandler) GetAllTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.repo.GetAllTemplates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// GetTemplate returns a specific template by ID
func (h *ImportTemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	template, err := h.repo.GetTemplateByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// CreateTemplate creates a new import template
func (h *ImportTemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.ImportTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.CreateTemplate(&template); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// UpdateTemplate updates an existing template
func (h *ImportTemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	var template models.ImportTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	template.ID = id
	if err := h.repo.UpdateTemplate(&template); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// DeleteTemplate deletes a template by ID
func (h *ImportTemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeleteTemplate(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestS3URL fetches JSON from S3 URL and returns the keys for mapping
func (h *ImportTemplateHandler) TestS3URL(w http.ResponseWriter, r *http.Request) {
	var request struct {
		S3URL string `json:"s3_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch JSON from S3 URL
	resp, err := http.Get(request.S3URL)
	if err != nil {
		http.Error(w, "Failed to fetch S3 URL: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "S3 URL returned status: "+resp.Status, http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse JSON to get keys
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Return the raw JSON for the frontend to parse
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
