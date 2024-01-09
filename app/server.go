package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const newline = "\r\n"

var storageDirectory *string

func main() {
	storageDirectory = flag.String("directory", "storage", "Storage directory")
	flag.Parse()

	host := "0.0.0.0"
	err := serve(host, 4221)
	if err != nil {
		fmt.Println("Error serving: ", err.Error())
		os.Exit(1)
	}
}

func serve(host string, port uint16) error {
	portString := strconv.FormatUint(uint64(port), 10)
	address := strings.Join([]string{host, portString}, ":")
	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Failed to bind to port " + portString)
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			return errors.New("error accepting connection")
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
	case "files":
		if len(pathComponents) < 2 {
			return Response{StatusCode: BadRequest}
		}

		filePath := strings.Join(pathComponents[1:], "/")
		directory := *storageDirectory
		path := filepath.Join(directory, filePath)

		if request.Verb == "POST" {
			// Read file contents from []byte and remove empty bytes
			fileContents := request.Body
			fileContents = bytes.Trim(fileContents, "\x00")

			os.WriteFile(path, fileContents, 0644)
			return Response{StatusCode: Created}
		} else if request.Verb == "GET" {
			// Reading file
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
		} else {
			return Response{StatusCode: NotFound}
		}
	default:
		return Response{StatusCode: NotFound}
	}
}
