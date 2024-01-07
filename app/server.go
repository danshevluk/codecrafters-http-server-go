package main

import (
	"errors"
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

	for {
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
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

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
	if len(request.Path) == 0 || request.Path[0] != '/' {
		return Response{StatusCode: BadRequest}
	}

	rawPathComponents := strings.Split(request.Path, "/")
	pathComponents := make([]string, 0, len(rawPathComponents))

	// Filter empty strings from path components
	for _, component := range rawPathComponents {
		if component == "" {
			continue
		}
		pathComponents = append(pathComponents, component)
	}

	if len(pathComponents) == 0 {
		pathComponents = append(pathComponents, "/")
	}

	switch pathComponents[0] {
	case "/":
		return Response{StatusCode: OK}
	case "echo":
		if len(pathComponents) < 2 {
			return Response{StatusCode: BadRequest}
		}
		return Response{
			StatusCode: OK,
		}.withBody(pathComponents[1], "text/plain")
	default:
		return Response{StatusCode: NotFound}
	}
}

// ==== Response

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

const (
	OK         = 200
	BadRequest = 400
	NotFound   = 404
)

func (r Response) statusText() string {
	switch r.StatusCode {
	case OK:
		return "OK"
	case BadRequest:
		return "Bad Request"
	case NotFound:
		return "Not Found"
	default:
		return ""
	}
}

const protocol = "HTTP/1.1"

func (r Response) encode() []byte {
	var responseString string

	// Status Line
	responseString += fmt.Sprintf("%s %v %s", protocol, r.StatusCode, r.statusText()) + newline

	// Headers
	for k, v := range r.Headers {
		responseString += fmt.Sprintf("%s: %s", k, v) + newline
	}
	responseString += newline

	var result []byte
	result = append(result, []byte(responseString)...)
	result = append(result, r.Body...)
	result = append(result, []byte(newline)...)

	fmt.Println("Encoded response: ")
	fmt.Println(string(result))
	return result
}

func (r Response) withBody(s string, contentType string) Response {
	byteBody := []byte(s)

	r.Body = byteBody

	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers["Content-Length"] = fmt.Sprintf("%v", len(byteBody))
	r.Headers["Content-Type"] = contentType
	return r
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

	return strings.TrimSpace(fields[1]), nil
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
