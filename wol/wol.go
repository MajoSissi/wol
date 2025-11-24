package wol

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// MagicPacket is a slice of 102 bytes containing the magic packet data.
type MagicPacket [102]byte

// NewMagicPacket creates a new MagicPacket for the given MAC address.
func NewMagicPacket(macAddr string) (*MagicPacket, error) {
	// Remove delimiters like :, - or .
	macAddr = strings.ReplaceAll(macAddr, ":", "")
	macAddr = strings.ReplaceAll(macAddr, "-", "")
	macAddr = strings.ReplaceAll(macAddr, ".", "")

	macBytes, err := hex.DecodeString(macAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %v", err)
	}

	if len(macBytes) != 6 {
		return nil, errors.New("invalid MAC address length")
	}

	var packet MagicPacket
	// First 6 bytes are 0xFF
	copy(packet[0:], []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})

	// Repeat MAC address 16 times
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], macBytes)
	}

	return &packet, nil
}

// Send sends the Magic Packet to the specified broadcast address and port.
// broadcastAddr should be in the form "ip:port", e.g., "255.255.255.255:9" or "192.168.1.255:9".
func (mp *MagicPacket) Send(broadcastAddr string) error {
	conn, err := net.Dial("udp", broadcastAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(mp[:])
	return err
}

// Wake sends a magic packet to the specified MAC address.
// If broadcastIP is empty, it broadcasts to all available IPv4 interfaces.
// It sends the packet multiple times with a delay between each send.
func Wake(macAddr, broadcastIP string, port int) error {
	mp, err := NewMagicPacket(macAddr)
	if err != nil {
		return err
	}

	var targets []string
	if broadcastIP != "" {
		targets = []string{fmt.Sprintf("%s:%d", broadcastIP, port)}
	} else {
		// Discover all broadcast addresses
		addrs, err := getBroadcastAddresses()
		if err != nil {
			return err
		}
		if len(addrs) == 0 {
			// Fallback to global broadcast if no interfaces found (unlikely)
			targets = []string{fmt.Sprintf("255.255.255.255:%d", port)}
		} else {
			for _, addr := range addrs {
				targets = append(targets, fmt.Sprintf("%s:%d", addr, port))
			}
		}
	}

	// Default to 5 times with 100ms interval
	for i := 0; i < 5; i++ {
		for _, target := range targets {
			// We ignore errors for individual targets to ensure we try all
			_ = mp.Send(target)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func getBroadcastAddresses() ([]string, error) {
	var list []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, ipnet, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ip.To4() != nil {
				ip4 := ip.To4()
				mask := ipnet.Mask
				broadcast := make(net.IP, len(ip4))
				for k := 0; k < len(ip4); k++ {
					broadcast[k] = ip4[k] | ^mask[k]
				}
				list = append(list, broadcast.String())
			}
		}
	}
	return list, nil
}
