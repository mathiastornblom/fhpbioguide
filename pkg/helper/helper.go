package helper

import (
	"net"
)

func GetPublicIPAddr() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	ip := ""
	if err != nil {
		ip = "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip = localAddr.IP.String()
	return ip
}
