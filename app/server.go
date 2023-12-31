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

	err = handleConnection(conn)
	if err != nil {
		fmt.Println("Error handling connection: ", err.Error())
		os.Exit(1)
	}
}

func handleConnection(conn net.Conn) error {
	readBuf := make([]byte, 1024)
	var err error
	_, err = conn.Read(readBuf)
	if err != nil {
		return err
	}
	requestLines := strings.Split(string(readBuf), newline)
	path := extractPath(&requestLines)

	if path == "/" {
		err = writeResponse(conn, "HTTP/1.1 200 OK"+newline+newline)
	} else {
		err = writeResponse(conn, "HTTP/1.1 404 Not Found"+newline+newline)
	}

	return err
}

func extractPath(requestLines *[]string) string {
	return strings.Split((*requestLines)[0], " ")[1]
}

func writeResponse(conn net.Conn, response string) error {
	_, err := conn.Write([]byte(response))
	if err != nil {
		return err
	}

	return nil
}
