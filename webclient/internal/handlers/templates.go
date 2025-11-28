package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FlavioMalvestitiJunior/bf-offers/webclient/internal/models"
	"github.com/FlavioMalvestitiJunior/bf-offers/webclient/internal/repository"
	"github.com/gorilla/mux"
)

type TemplateHandler struct {
	templateRepo *repository.TemplateRepository
}

func NewTemplateHandler(templateRepo *repository.TemplateRepository) *TemplateHandler {
	return &TemplateHandler{templateRepo: templateRepo}
}

// GetAllTemplates returns all message templates
func (h *TemplateHandler) GetAllTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.templateRepo.GetAllTemplates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// GetTemplate returns a specific template by ID
func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	template, err := h.templateRepo.GetTemplateByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// CreateTemplate creates a new message template
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.MessageTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.templateRepo.CreateTemplate(&template); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// UpdateTemplate updates an existing template
func (h *TemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	var template models.MessageTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	template.ID = id
	if err := h.templateRepo.UpdateTemplate(&template); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// DeleteTemplate deletes a template
func (h *TemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	if err := h.templateRepo.DeleteTemplate(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
