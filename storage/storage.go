package storage

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"regexp"
	"sync"
)

type SubDevice struct {
	MAC         string `json:"mac"`
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	BroadcastIP string `json:"broadcast_ip"`
	Remark      string `json:"remark"`
}

type Device struct {
	Name        string      `json:"name"`
	MAC         string      `json:"mac,omitempty"`
	IP          string      `json:"ip,omitempty"`
	Port        int         `json:"port,omitempty"`
	BroadcastIP string      `json:"broadcast_ip,omitempty"`
	SubDevices  []SubDevice `json:"sub_devices,omitempty"`
	PingMode    string      `json:"ping_mode,omitempty"` // "any" or "all"
}

type Store struct {
	mu               sync.RWMutex
	filename         string
	Port             int      `json:"port"`
	LogDir           string   `json:"log_dir"`
	LogRetentionDays int      `json:"log_retention_days"`
	Devices          []Device `json:"devices"`
}

func NewStore(filename string) (*Store, error) {
	s := &Store{
		filename:         filename,
		Port:             8888, // Default port
		LogDir:           "./logs",
		LogRetentionDays: 3,
		Devices:          []Device{},
	}
	if err := s.Load(); err != nil {
		// If file doesn't exist, create it with defaults
		if os.IsNotExist(err) {
			if err := s.Save(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	// Ensure defaults if loaded file didn't have them
	if s.Port == 0 {
		s.Port = 8888
	}
	if s.LogDir == "" {
		s.LogDir = "./logs"
	}
	if s.LogRetentionDays == 0 {
		s.LogRetentionDays = 3
	}
	return s, nil
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, s)
}

func (s *Store) saveInternal() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filename, data, 0644)
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveInternal()
}

func (s *Store) AddDevice(d Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := d.Validate(); err != nil {
		return err
	}

	// Clear top-level fields if SubDevices is present to avoid duplication
	if len(d.SubDevices) > 0 {
		d.MAC = ""
		d.IP = ""
		d.Port = 0
		d.BroadcastIP = ""
	}

	for _, dev := range s.Devices {
		if dev.Name == d.Name {
			return errors.New("device with this name already exists")
		}
	}

	s.Devices = append(s.Devices, d)
	return s.saveInternal()
}

func (s *Store) ReorderDevices(names []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(names) != len(s.Devices) {
		return errors.New("device count mismatch")
	}

	deviceMap := make(map[string]Device)
	for _, d := range s.Devices {
		deviceMap[d.Name] = d
	}

	newDevices := make([]Device, 0, len(s.Devices))
	for _, name := range names {
		if d, ok := deviceMap[name]; ok {
			newDevices = append(newDevices, d)
		} else {
			return errors.New("device not found: " + name)
		}
	}

	s.Devices = newDevices
	return s.saveInternal()
}

func (s *Store) UpdateDevice(oldName string, d Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := d.Validate(); err != nil {
		return err
	}

	// Clear top-level fields if SubDevices is present to avoid duplication
	if len(d.SubDevices) > 0 {
		d.MAC = ""
		d.IP = ""
		d.Port = 0
		d.BroadcastIP = ""
	}

	// If name is changing, check for conflict
	if oldName != d.Name {
		for _, dev := range s.Devices {
			if dev.Name == d.Name {
				return errors.New("device with this name already exists")
			}
		}
	}

	for i, dev := range s.Devices {
		if dev.Name == oldName {
			s.Devices[i] = d
			return s.saveInternal()
		}
	}
	return errors.New("device not found")
}

func (s *Store) DeleteDevice(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newDevices := []Device{}
	found := false
	for _, d := range s.Devices {
		if d.Name != name {
			newDevices = append(newDevices, d)
		} else {
			found = true
		}
	}
	
	if !found {
		return errors.New("device not found")
	}

	s.Devices = newDevices
	return s.saveInternal()
}

func (s *Store) GetDevice(name string) (Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.Devices {
		if d.Name == name {
			return d, true
		}
	}
	return Device{}, false
}

func (s *Store) GetAll() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy
	result := make([]Device, len(s.Devices))
	copy(result, s.Devices)
	return result
}

func (s *Store) GetPort() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Port
}

func isValidMAC(mac string) bool {
	_, err := net.ParseMAC(mac)
	return err == nil
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func isValidHostname(host string) bool {
	if len(host) > 255 {
		return false
	}
	// Simple regex for hostname (RFC 1123)
	re := regexp.MustCompile(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`)
	return re.MatchString(host)
}

func isValidHostOrIP(host string) bool {
	if host == "" {
		return true // Optional
	}
	if isValidIP(host) {
		return true
	}
	return isValidHostname(host)
}

func (sd *SubDevice) Validate() error {
	if !isValidMAC(sd.MAC) {
		return errors.New("invalid MAC address: " + sd.MAC)
	}
	if !isValidHostOrIP(sd.IP) {
		return errors.New("invalid IP or Hostname: " + sd.IP)
	}
	if sd.Port < 1 || sd.Port > 65535 {
		return errors.New("invalid port number")
	}
	if sd.BroadcastIP != "" && !isValidIP(sd.BroadcastIP) {
		return errors.New("invalid broadcast IP: " + sd.BroadcastIP)
	}
	return nil
}

func (d *Device) Validate() error {
	if d.Name == "" {
		return errors.New("device name is required")
	}
	if len(d.SubDevices) > 0 {
		for _, sd := range d.SubDevices {
			if err := sd.Validate(); err != nil {
				return err
			}
		}
	} else {
		// Fallback for single device structure if used directly
		sd := SubDevice{
			MAC:         d.MAC,
			IP:          d.IP,
			Port:        d.Port,
			BroadcastIP: d.BroadcastIP,
		}
		if err := sd.Validate(); err != nil {
			return err
		}
	}
	return nil
}
