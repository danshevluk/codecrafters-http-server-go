package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const newline = "\r\n"

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	readBuf := make([]byte, 1024)
	_, err = conn.Read(readBuf)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
		os.Exit(1)
	}
	requestLines := strings.Split(string(readBuf), newline)
	path := extractPath(&requestLines)

	if path == "/" {
		writeResponse(conn, "HTTP/1.1 200 OK"+newline+newline)
	} else {
		writeResponse(conn, "HTTP/1.1 404 Not Found"+newline+newline)
	}
}

func extractPath(requestLines *[]string) string {
	return strings.Split((*requestLines)[0], " ")[1]
}

func writeResponse(conn net.Conn, response string) {
	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing to connection: ", err.Error())
		os.Exit(1)
	}
}
