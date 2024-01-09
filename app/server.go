package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const newline = "\r\n"

var storageDirectory *string

func main() {
	// Parse CLI flags
	storageDirectory = flag.String("directory", "storage", "Storage directory")
	flag.Parse()

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

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	err := processConnection(conn)
	if err != nil {
		fmt.Println("Error handling connection: ", err.Error())
	}
}

func processConnection(conn net.Conn) error {
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
	fmt.Println("Request: ")
	fmt.Println(request)
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
		}.withStringBody(strings.Join(pathComponents[1:], "/"), "text/plain")
	case "user-agent":
		if request.Headers == nil || request.Headers["User-Agent"] == "" {
			return Response{StatusCode: BadRequest}
		}

		userAgent := request.Headers["User-Agent"]
		return Response{
			StatusCode: OK,
		}.withStringBody(userAgent, "text/plain")
	case "file":
		if len(pathComponents) < 2 {
			return Response{StatusCode: BadRequest}
		}

		filePath := strings.Join(pathComponents[1:], "/")
		directory := *storageDirectory
		path := filepath.Join(directory, filePath)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return Response{StatusCode: NotFound}
		}

		fileContents, err := os.ReadFile(path)
		if err != nil {
			return Response{StatusCode: ServerErr}
		}

		return Response{
			StatusCode: OK,
		}.withBody(fileContents, "application/octet-stream")
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
	ServerErr  = 500
)

func (r Response) statusText() string {
	switch r.StatusCode {
	case OK:
		return "OK"
	case BadRequest:
		return "Bad Request"
	case NotFound:
		return "Not Found"
	case ServerErr:
		return "Internal Server Error"
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
	if r.Body != nil {
		result = append(result, r.Body...)
		result = append(result, []byte(newline)...)
	}

	fmt.Println("Encoded response: ")
	fmt.Println(string(result))
	return result
}

func (r Response) withStringBody(s string, contentType string) Response {
	return r.withBody([]byte(s), contentType)
}

func (r Response) withBody(data []byte, contentType string) Response {
	byteBody := data

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
	requestHeadersData := statusLineAndHeaders[1:]
	request.Headers, err = extractHeaders(requestHeadersData)
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
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		fields := strings.SplitN(line, ": ", 2)
		if len(fields) < 2 {
			continue
		}

		headers[fields[0]] = fields[1]
	}

	return headers, nil
}
