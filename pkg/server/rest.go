package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/ebobo/modem_prod_go/pkg/model"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func (s *Server) startHTTP() error {
	m := mux.NewRouter()

	// Add CORS
	cors := cors.New(cors.Options{
		AllowCredentials: true,
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		MaxAge:           31,
		Debug:            false,
	})

	// This is where you add other stuff you want to map in the mux

	// Config endpoint
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, Welcome to the Modem Production Server !")
	}).Methods("GET")

	// Get All modems
	m.HandleFunc("/api/v1/modems", s.GetListmodems).Methods("GET")

	// Add new modem
	m.HandleFunc("/api/v1/modem", s.Addmodem).Methods("POST")

	// Get modem by MacAddress
	m.HandleFunc("/api/v1/modem/{mac}", s.Getmodem).Methods("GET")

	// Update modem by MacAddress can use PATCH or PUT
	m.HandleFunc("/api/v1/modem/{mac}", s.Updatemodem).Methods("PUT")

	// Delete modem by MacAddress
	m.HandleFunc("/api/v1/modem/{mac}", s.Deletemodem).Methods("DELETE")

	// Update modem state by MacAddress
	m.HandleFunc("/api/v1/modem/{mac}/state", s.SetModemState).Methods("PUT")

	// Update modem upgrade progress by MacAddress
	m.HandleFunc("/api/v1/modem/{mac}/progress", s.SetModemProgress).Methods("PUT")

	httpServer := &http.Server{
		Addr:              s.httpListenAddr,
		Handler:           handlers.ProxyHeaders(cors.Handler(m)),
		ReadTimeout:       (10 * time.Second),
		ReadHeaderTimeout: (8 * time.Second),
		WriteTimeout:      (45 * time.Second),
	}

	// Set up shutdown handler
	go func() {
		<-s.ctx.Done()
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Printf("error shutting down HTTP interface '%s': %v", s.httpListenAddr, err)
		}
	}()

	// Start HTTP server
	go func() {
		log.Printf("starting HTTP interface '%s'", s.httpListenAddr)

		// This isn't entirely true and really represents a race condition, but
		// doing this properly is a pain in the neck.
		s.httpStarted.Done()

		err := httpServer.ListenAndServe()
		if err == http.ErrServerClosed {
			err = errors.New("")
		}

		log.Printf("HTTP interface '%s' down %v", s.httpListenAddr, err)
		s.httpStopped.Done()
	}()

	return nil
}

func (s *Server) GetListmodems(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	modems, err := s.db.ListModems()
	if err != nil {
		log.Printf("failed to get modems %v", err)
		http.Error(w, "failed to get modems", http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(modems)
}

func (s *Server) Getmodem(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	modemMac := mux.Vars(r)["mac"]
	log.Printf("modemMacAdress: %s", modemMac)
	modem, err := s.db.GetModem(modemMac)

	if err != nil {
		log.Printf("failed to get modem %v by mac address %s", err, modemMac)
		http.Error(w, "failed to get modem", http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(modem)
}

func (s *Server) Addmodem(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var newmodem model.Modem

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter data with the modem detail in order to update")
	}
	json.Unmarshal(reqBody, &newmodem)

	err = s.db.AddModem(newmodem)

	if err != nil {
		log.Printf("failed to add modem %v", err)
		http.Error(w, "failed to add modem", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newmodem)
}

func (s *Server) Updatemodem(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	macAddress := mux.Vars(r)["mac"]
	modem, err := s.db.GetModem(macAddress)

	if err != nil {
		log.Printf("failed to get modem %v by mac address %s", err, macAddress)
		http.Error(w, "failed to update modem", http.StatusBadRequest)
		return
	}

	var updateModem model.Modem

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter data with the modem detail in order to update")
	}
	json.Unmarshal(reqBody, &updateModem)

	// update modem

	if updateModem.Model != "" {
		modem.Model = updateModem.Model
	}

	modem.State = updateModem.State
	modem.Upgraded = updateModem.Upgraded

	if updateModem.Firmware != "" {
		modem.Firmware = updateModem.Firmware
	}

	log.Println("update modem", modem.MacAddress, modem.Model, modem.State, modem.Firmware)

	err = s.db.UpdateModem(modem)

	if err != nil {
		log.Printf("failed to get modem %v by id %s", err, macAddress)
		http.Error(w, "failed to update modem", http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(modem)
}

func (s *Server) Deletemodem(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	macAddress := mux.Vars(r)["mac"]
	err := s.db.DeleteModem(macAddress)

	if err != nil {
		log.Printf("failed to delete modem %v by mac address %s", err, macAddress)
		http.Error(w, "failed to delete modem", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) SetModemState(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	macAddress := mux.Vars(r)["mac"]

	// Read the new state from the request body.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("failed to read request body: %v", err)
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var newState struct {
		State int `json:"state"`
	}
	if err := json.Unmarshal(body, &newState); err != nil {
		log.Printf("failed to unmarshal request body: %v", err)
		http.Error(w, "failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	// Update the state in the database.
	if err := s.db.SetModemState(macAddress, newState.State); err != nil {
		log.Printf("failed to set modem state: %v", err)
		http.Error(w, "failed to set modem state", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) SetModemProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	macAddress := mux.Vars(r)["mac"]

	// Read the new progress from the request body.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("failed to read request body: %v", err)
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var newProgress struct {
		Progress int `json:"progress"`
	}
	if err := json.Unmarshal(body, &newProgress); err != nil {
		log.Printf("failed to unmarshal request body: %v", err)
		http.Error(w, "failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	// Update the progress in the database.
	if err := s.db.SetModemUpgradeProgress(macAddress, newProgress.Progress); err != nil {
		log.Printf("failed to set modem upgrade progress: %v", err)
		http.Error(w, "failed to set modem upgrade progress", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
