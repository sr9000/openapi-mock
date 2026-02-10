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

	var pet gen.NewPet
	json.NewDecoder(r.Body).Decode(&pet)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(gen.Pet{Id: 123, Name: pet.Name, Tag: pet.Tag})
}

func (h *PetsHandlers) DeletePet(w http.ResponseWriter, r *http.Request, petId int64) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] DeletePet petId=%d", reqID, petId)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PetsHandlers) GetPetById(w http.ResponseWriter, r *http.Request, petId int64) {
	if h.EnableLogging {
		reqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] GetPetById petId=%d", reqID, petId)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gen.Pet{Id: petId, Name: "Mock Pet"})
}
