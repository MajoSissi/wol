package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"

	"wol/logger"
	"wol/storage"
	"wol/wol"
)

var (
	//go:embed static
	staticFiles embed.FS

	store      *storage.Store
	daemonMode = flag.Bool("d", false, "Run in background (daemon mode)")
	killMode   = flag.Bool("k", false, "Kill the running daemon (Linux only)")
)

func main() {
	flag.Parse()

	var err error
	store, err = storage.NewStore("wol.json")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	if err := logger.Init(store.LogDir, store.LogRetentionDays); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Setup HTTP handlers
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create static file system: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(staticFS)))
	http.HandleFunc("/api/devices", handleDevices)
	http.HandleFunc("/api/devices/reorder", handleDeviceReorder)
	http.HandleFunc("/api/devices/", handleDeviceAction) // For update/delete
	http.HandleFunc("/api/wake/", handleWake)
	http.HandleFunc("/api/ping/", handlePing)
	http.HandleFunc("/api/logs", handleLogs)

	// Delegate to platform specific run logic
	runPlatformSpecific()
}

func startServer() {
	port := store.GetPort()
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server started at http://localhost:%d\n", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Error("System", fmt.Sprintf("Server failed: %v", err))
		log.Fatal(err)
	}
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		devices := store.GetAll()
		json.NewEncoder(w).Encode(devices)
	case http.MethodPost:
		var d storage.Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if d.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}
		if err := store.AddDevice(d); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.Info(d.Name, "Device added")
		json.NewEncoder(w).Encode(d)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleDeviceReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var names []string
	if err := json.NewDecoder(r.Body).Decode(&names); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.ReorderDevices(names); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeviceAction(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/api/devices/"):]
	if name == "" {
		http.Error(w, "Name required", http.StatusBadRequest)
		return
	}
	
	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		http.Error(w, "Invalid name encoding", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var d storage.Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// We pass the old name (from URL) and the new device object (from body)
		if err := store.UpdateDevice(decodedName, d); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.Info(d.Name, fmt.Sprintf("Device updated (old name: %s)", decodedName))
		json.NewEncoder(w).Encode(d)
	case http.MethodDelete:
		if err := store.DeleteDevice(decodedName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.Info(decodedName, "Device deleted")
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWake(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/api/wake/"):]
	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		http.Error(w, "Invalid name encoding", http.StatusBadRequest)
		return
	}

	device, found := store.GetDevice(decodedName)
	if !found {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	targetPort := device.Port
	if targetPort == 0 {
		targetPort = 9
	}

	// Log the attempt
	if len(device.SubDevices) > 0 {
		logger.Info(device.Name, fmt.Sprintf("Sending WOL packets to group (%d devices)...", len(device.SubDevices)))
		
		var errs []string
		for i, sub := range device.SubDevices {
			targetPort := sub.Port
			if targetPort == 0 {
				targetPort = 9
			}
			targetDesc := sub.BroadcastIP
			if targetDesc == "" {
				targetDesc = "all interfaces"
			}
			
			if err := wol.Wake(sub.MAC, sub.BroadcastIP, targetPort); err != nil {
				errMsg := fmt.Sprintf("Device %d (%s): %v", i+1, sub.MAC, err)
				errs = append(errs, errMsg)
				logger.Error(device.Name, errMsg)
			}
		}
		
		if len(errs) > 0 {
			// If some failed, we still consider it a partial success or just log errors
			// For HTTP response, if all failed, maybe error?
			// Let's just return OK but log errors.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Group wake completed with %d errors", len(errs))))
			return
		}
		
		logger.Info(device.Name, "Group wake completed successfully")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Group wake completed"))
		return
	}

	targetDesc := device.BroadcastIP
	if targetDesc == "" {
		targetDesc = "all interfaces"
	}
	logger.Info(device.Name, fmt.Sprintf("Sending WOL packets to %s:%d...", targetDesc, targetPort))

	// Wake function now handles repeated sending internally (5 times, 100ms interval)
	// If BroadcastIP is empty, it iterates over all IPv4 interfaces.
	if err := wol.Wake(device.MAC, device.BroadcastIP, targetPort); err != nil {
		errMsg := fmt.Sprintf("Failed to send WOL packet: %v", err)
		logger.Error(device.Name, errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	successMsg := fmt.Sprintf("Magic packets sent to %s:%d", targetDesc, targetPort)
	logger.Info(device.Name, successMsg)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successMsg))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/api/ping/"):]
	decodedName, err := url.QueryUnescape(name)
	if err != nil {
		http.Error(w, "Invalid name encoding", http.StatusBadRequest)
		return
	}

	device, found := store.GetDevice(decodedName)
	if !found {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	if len(device.SubDevices) > 0 {
		// For groups, we check if the FIRST device is online
		online := false
		if device.SubDevices[0].IP != "" {
			online = ping(device.SubDevices[0].IP)
		}
		json.NewEncoder(w).Encode(map[string]bool{"online": online})
		return
	}

	if device.IP == "" {
		http.Error(w, "Device has no IP address", http.StatusBadRequest)
		return
	}

	online := ping(device.IP)
	json.NewEncoder(w).Encode(map[string]bool{"online": online})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	device := r.URL.Query().Get("device")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	logs, err := logger.GetLogs(device, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(logs)
}
