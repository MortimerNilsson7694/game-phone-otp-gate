package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/example/game-phone-otp/internal/game"
	"github.com/example/game-phone-otp/internal/infrai"
)

type server struct {
	phone *infrai.Client
	store *game.Store
}

func main() {
	client, err := infrai.NewClient(infrai.DefaultBaseURL, os.Getenv("INFRAI_API_KEY"), nil)
	if err != nil {
		log.Fatal(err)
	}
	s := &server{phone: client, store: game.NewStore()}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code", s.sendCode)
	mux.HandleFunc("POST /login/verify", s.verifyCode)
	mux.HandleFunc("GET /login/delivery/{id}", s.delivery)
	mux.HandleFunc("POST /assets", s.addAsset)
	mux.HandleFunc("POST /events", s.addEvent)
	mux.HandleFunc("GET /moderation", s.queue)
	log.Printf("game-login listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) sendCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone  string `json:"phone"`
		Locale string `json:"locale"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := s.phone.SendLoginCode(r.Context(), in.Phone, in.Locale)
	respondResult(w, result, err)
}

func (s *server) verifyCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := s.phone.VerifyLoginCode(r.Context(), in.Phone, in.Code)
	respondResult(w, result, err)
}

func (s *server) delivery(w http.ResponseWriter, r *http.Request) {
	result, err := s.phone.DeliveryStatus(r.Context(), r.PathValue("id"))
	respondResult(w, result, err)
}

func (s *server) addAsset(w http.ResponseWriter, r *http.Request) {
	var asset game.Asset
	if !decode(w, r, &asset) {
		return
	}
	state, err := s.store.AddAsset(asset)
	respondResult(w, map[string]string{"state": state}, err)
}

func (s *server) addEvent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		StartsAt  string `json:"starts_at"`
		CreatedBy string `json:"created_by"`
	}
	if !decode(w, r, &in) {
		return
	}
	startsAt, err := time.Parse(time.RFC3339, in.StartsAt)
	if err == nil {
		err = s.store.AddEvent(game.LiveEvent{ID: in.ID, Name: in.Name, StartsAt: startsAt, CreatedBy: in.CreatedBy})
	}
	respondResult(w, map[string]string{"state": "scheduled"}, err)
}

func (s *server) queue(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ModerationQueue())
}

func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return false
	}
	return true
}

func respondResult(w http.ResponseWriter, value any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, value)
		return
	}
	status := http.StatusBadRequest
	var apiErr *infrai.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		status = apiErr.HTTPStatus
	} else if errors.As(err, &apiErr) || strings.HasPrefix(err.Error(), "send request:") || strings.HasPrefix(err.Error(), "decode response") {
		status = http.StatusBadGateway
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
