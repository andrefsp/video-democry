package netutils

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var prefixes = []string{"en", "eth", "wl"}

func GetRelayAddr() (string, error) {

	ifaces, err := GetIFaces()
	if err != nil {
		return "", err
	}

	if len(ifaces) == 0 {
		return "", errors.New("No interface found")
	}

	iface := ifaces[0]
	if err != nil {
		return "", err
	}

	iFaceAddrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for a := range iFaceAddrs {
		ip := net.ParseIP(strings.Split(iFaceAddrs[a].String(), "/")[0])
		if ip.To4() != nil {
			return ip.To4().String(), nil
		}
	}

	return "", errors.New("No interface address found")
}

func GetIFaces() ([]net.Interface, error) {
	hostIFaces, err := net.Interfaces()
	if err != nil {
		fmt.Print(fmt.Errorf("localAddresses: %+v\n", err.Error()))
		return nil, err
	}

	ifaces := []net.Interface{}

	for i := range hostIFaces {
		if hostIFaces[i].Flags&net.FlagLoopback == net.FlagLoopback {
			continue
		}

		for _, prefix := range prefixes {
			if strings.HasPrefix(hostIFaces[i].Name, prefix) {
				ifaces = append(ifaces, hostIFaces[i])
				break
			}
		}
	}

	return ifaces, nil
}
