package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	marketplacechat "example.com/marketplace-chat"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	service := marketplacechat.HandoffService{Realtime: &marketplacechat.Client{APIKey: apiKey}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders/handoff", func(w http.ResponseWriter, r *http.Request) {
		var request marketplacechat.HandoffRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		result, err := service.Complete(r.Context(), request)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, marketplacechat.ErrHandoffNotReady) {
				status = http.StatusConflict
			}
			var apiErr *marketplacechat.APIError
			if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
				status = apiErr.HTTPStatus
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	address := ":8080"
	log.Printf("marketplace chat listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
