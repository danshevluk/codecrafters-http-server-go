package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

const newline = "\r\n"
const defaultBufSize = 4096

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
	// Read bytes
	readBuf := make([]byte, 1024)
	_, err := conn.Read(readBuf)
	if err != nil {
		return err
	}

	// Parse request
	request, err := parseRequest(readBuf)
	if request == nil || err != nil {
		return err
	}
	unwrapedRequest := *request

	// Write response
	response := handleRequest(unwrapedRequest)
	err = response.write(conn)
	if err != nil {
		return err
	}

	// Connection handled successfully
	return nil
}

func handleRequest(request Request) Response {
	switch request.Path {
	case "/":
		return Response{StatusCode: OK}
	default:
		return Response{StatusCode: NotFound}
	}
}

// ==== Response

type Response struct {
	StatusCode int
	Headers    map[string]string
}

const (
	OK       = 200
	NotFound = 404
)

func (r Response) statusText() string {
	switch r.StatusCode {
	case OK:
		return "OK"
	case NotFound:
		return "Not Found"
	default:
		return ""
	}
}

const protocol = "HTTP/1.1"

func (r Response) encode() []byte {
	var response string

	// Status Line
	response += fmt.Sprintf("%s %v %s", protocol, r.StatusCode, r.statusText()) + newline

	// Headers
	for k, v := range r.Headers {
		response += fmt.Sprintf("%s: %s", k, v) + newline
	}

	return []byte(response)
}

func (r Response) write(conn net.Conn) error {
	_, err := conn.Write(r.encode())
	if err != nil {
		return err
	}

	return nil
}

// ==== Request

type Request struct {
	Path    string
	Headers map[string]string
}

func parseRequest(requestBytes []byte) (*Request, error) {
	requestParts := strings.SplitN(string(requestBytes), newline+newline, 2)
	if len(requestParts) == 0 {
		return nil, errors.New("request is empty")
	}

	statusLineAndHeaders := strings.Split(requestParts[0], newline)
	if len(statusLineAndHeaders) == 0 {
		return nil, errors.New("status line not found")
	}

	var request Request
	var err error

	statusLine := statusLineAndHeaders[0]
	request.Path, err = extractPath(statusLine)
	if err != nil {
		return nil, err
	}
	if len(statusLineAndHeaders) < 2 {
		return &request, nil
	}
	requestHeaders := statusLineAndHeaders[1:]

	var headers []string
	for i, line := range requestHeaders {
		if strings.TrimSpace(line) == "" {
			headers = requestHeaders[:i]
			break
		}
	}

	request.Headers, err = extractHeaders(headers)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func extractPath(statusLine string) (string, error) {
	fields := strings.Fields(statusLine)
	if len(fields) < 2 {
		return "", errors.New("fields not found")
	}

	return fields[1], nil
}

func extractHeaders(headerLines []string) (map[string]string, error) {
	headers := make(map[string]string)

	for _, line := range headerLines {
		if strings.TrimSpace(line) == "" {
			break
		}

		fields := strings.SplitN(line, ":", 2)
		if len(fields) < 2 {
			continue
		}

		headers[fields[0]] = fields[1]
	}

	return headers, nil
}
