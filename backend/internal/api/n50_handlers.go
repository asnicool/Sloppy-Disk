package api

import (
	"log"
	"net/http"

	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/n50"

	"github.com/gorilla/mux"
)

// RegisterN50Routes registers N50-specific routes
func RegisterN50Routes(r *mux.Router) {
	api := r.PathPrefix("/api").Subrouter()

	// N50 Status and Control
	api.HandleFunc("/n50/status", GetN50Status).Methods("GET")
	api.HandleFunc("/n50/power/on", N50PowerOn).Methods("POST")
	api.HandleFunc("/n50/power/off", N50PowerOff).Methods("POST")
	api.HandleFunc("/n50/input/{input}", N50SetInput).Methods("POST")
	api.HandleFunc("/n50/inputs", GetN50AvailableInputs).Methods("GET")
}

// GetN50Status returns the current status of the N50 component
func GetN50Status(w http.ResponseWriter, r *http.Request) {
	if !n50.IsEnabled() {
		SendJSON(w, models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"enabled":     false,
				"isConnected": false,
				"message":     "N50 is not enabled or configured",
			},
		})
		return
	}

	client := n50.GetClient()
	status, err := client.GetStatus()
	if err != nil {
		log.Printf("[N50] Error getting status: %v", err)
		SendJSON(w, models.APIResponse{
			Success: false,
			Error:   err.Error(),
			Data: map[string]interface{}{
				"enabled":     n50.IsEnabled(),
				"isConnected": false,
				"error":       err.Error(),
			},
		})
		return
	}

	SendJSON(w, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"enabled":        n50.IsEnabled(),
			"isConnected":    status.IsConnected,
			"powerStatus":    status.PowerStatus,
			"currentInput":   status.CurrentInput,
			"powerRaw":       status.PowerRaw,
			"inputRaw":       status.InputRaw,
			"configuredInput": n50.GetConfiguredInput(),
		},
	})
}

// N50PowerOn powers on the N50 component
func N50PowerOn(w http.ResponseWriter, r *http.Request) {
	if !n50.IsEnabled() {
		SendError(w, http.StatusServiceUnavailable, "N50 is not enabled or configured")
		return
	}

	client := n50.GetClient()
	if err := client.PowerUp(); err != nil {
		log.Printf("[N50] Error powering on: %v", err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSON(w, models.APIResponse{Success: true, Data: map[string]string{"message": "N50 powered on"}})
}

// N50PowerOff puts the N50 component in standby
func N50PowerOff(w http.ResponseWriter, r *http.Request) {
	if !n50.IsEnabled() {
		SendError(w, http.StatusServiceUnavailable, "N50 is not enabled or configured")
		return
	}

	client := n50.GetClient()
	if err := client.StandBy(); err != nil {
		log.Printf("[N50] Error powering off: %v", err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSON(w, models.APIResponse{Success: true, Data: map[string]string{"message": "N50 put in standby"}})
}

// N50SetInput sets the input source of the N50
func N50SetInput(w http.ResponseWriter, r *http.Request) {
	if !n50.IsEnabled() {
		SendError(w, http.StatusServiceUnavailable, "N50 is not enabled or configured")
		return
	}

	vars := mux.Vars(r)
	input := vars["input"]

	client := n50.GetClient()
	if err := client.SetInput(input); err != nil {
		log.Printf("[N50] Error setting input to %s: %v", input, err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSON(w, models.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Input set to " + input, "input": input},
	})
}

// GetN50AvailableInputs returns the list of available input sources
func GetN50AvailableInputs(w http.ResponseWriter, r *http.Request) {
	inputs := n50.GetAvailableInputs()
	SendJSON(w, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"inputs":          inputs,
			"configuredInput": n50.GetConfiguredInput(),
		},
	})
}

// checkN50BeforePlayback checks and ensures N50 is ready before playback
// This function is called by the playback control handlers
func checkN50BeforePlayback() error {
	// Check if N50 is enabled
	if !n50.IsEnabled() {
		return nil // N50 not enabled, no check needed
	}

	// Check if we should ignore N50 for starting playback
	if n50.ShouldIgnoreOnStart() {
		log.Printf("[N50] Ignoring N50 check due to configuration")
		return nil
	}

	// Check if auto control is enabled
	if !n50.ShouldAutoControl() {
		return nil // Auto control disabled, skip check
	}

	client := n50.GetClient()
	wasReady, err := client.EnsureReady()
	if err != nil {
		return err
	}

	if !wasReady {
		log.Printf("[N50] N50 was not ready, powered on and/or changed input")
	}

	return nil
}

// wrapPlaybackWithN50Check wraps a playback handler with N50 check
func wrapPlaybackWithN50Check(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check N50 before playback
		if err := checkN50BeforePlayback(); err != nil {
			log.Printf("[N50] Error ensuring N50 is ready: %v", err)
			// Don't fail the playback, just log the error
			// This ensures music can still play even if N50 has issues
		}

		// Call the original handler
		handler(w, r)
	}
}
