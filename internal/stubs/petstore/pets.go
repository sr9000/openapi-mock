package petstore

import (
	"encoding/json"
	"log"
	"net/http"
	gen "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/ctxkeys"
)

type PetsHandlers struct {
	EnableLogging bool
}

func NewPetsHandlers(enableLogging bool) *PetsHandlers {
	return &PetsHandlers{EnableLogging: enableLogging}
}

func (h *PetsHandlers) ListPets(w http.ResponseWriter, r *http.Request, params gen.ListPetsParams) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] ListPets", reqID)
	}

	// CUSTOM: Return realistic mock pets
	pets := []gen.Pet{
		{Id: 1, Name: "Fluffy"},
		{Id: 2, Name: "Buddy"},
		{Id: 3, Name: "Max"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pets)
}

func (h *PetsHandlers) CreatePet(w http.ResponseWriter, r *http.Request) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] CreatePet", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *PetsHandlers) DeletePet(w http.ResponseWriter, r *http.Request, petId int64) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] DeletePet", reqID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PetsHandlers) GetPetById(w http.ResponseWriter, r *http.Request, petId int64) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] GetPetById", reqID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
