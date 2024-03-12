package helper

import (
	"net"
)

// GetPublicIPAddr attempts to determine the public IP address of the current machine.
func GetPublicIPAddr() string {
	// Dial a UDP connection to Google's DNS server.
	// This is used because UDP does not actually establish a connection;
	// it just sets up the networking infrastructure to send packets.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// If there's an error (e.g., network is down), return localhost IP.
		// This assumes that failure to connect means we're offline or otherwise unable to determine the public IP.
		return "127.0.0.1"
	}
	// Ensure the connection is closed once we're done with it to free up system resources.
	defer conn.Close()

	// Extract the local address from the connection. This will be the address that
	// would be used to send packets to the destination set up in the Dial call.
	localAddr := conn.LocalAddr().(*net.UDPAddr) // Type assertion to *net.UDPAddr since LocalAddr returns net.Addr.

	// Convert the IP address to a string and return it.
	// This is the address that the rest of the internet would see this machine as.
	return localAddr.IP.String()
}
