package network

import "net"

// MapInterfaces provides a map of interfaces to IP addresses.
func MapInterfaces() (map[string][]string, error) {
	m := make(map[string][]string)
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, adapter := range interfaces {
		addresses, err := adapter.Addrs()
		if err != nil {
			return nil, err
		}

		var ips []string
		for _, address := range addresses {
			ips = append(ips, address.String())
		}
		m[adapter.Name] = ips
	}

	return m, nil
}
