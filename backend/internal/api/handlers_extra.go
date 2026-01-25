package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"

	"github.com/gorilla/mux"
)

func MovePlaylistTrack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From   int `json:"from"`
		To     int `json:"to"`
		Length int `json:"length"` // Optional, defaults to 1
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	log.Printf("[API] MovePlaylistTrack request: From=%d, To=%d, Length=%d", req.From, req.To, req.Length)

	var err error
	if req.Length > 1 {
		// Move a range [From, From+Length)
		// Note: MPD move range end is exclusive, so From+Length is correct for [From, From+Length)
		log.Printf("[API] Calling MPD MoveRange: start=%d, end=%d, to=%d", req.From, req.From+req.Length, req.To)
		err = mpd.GetClient().MoveRange(req.From, req.From+req.Length, req.To)
	} else {
		log.Printf("[API] Calling MPD Move: from=%d, to=%d", req.From, req.To)
		err = mpd.GetClient().Move(req.From, req.To)
	}

	if err != nil {
		log.Printf("[API] Move failed: %v", err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[API] Move successful")
	SendJSON(w, models.APIResponse{Success: true})
}

func RemovePlaylistTrack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	posStr, _ := url.PathUnescape(vars["pos"])
	pos := parseInt(posStr) // Helper already exists in handlers.go, but it's private.
	// Wait, if it's private in the same package, it IS accessible!
	// parseint is in handlers.go, package api. This file is package api. So it should work.

	if err := mpd.GetClient().Delete(pos); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}
