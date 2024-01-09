package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type HTTPServer struct {
	Host   string
	Port   uint16
	Router HTTPRouter
}

func (s *HTTPServer) Serve() error {
	portString := strconv.FormatUint(uint64(s.Port), 10)
	address := strings.Join([]string{s.Host, portString}, ":")
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

		go s.handleConnection(conn)
	}
}

func (s *HTTPServer) handleConnection(conn net.Conn) {
	err := s.Router.processConnection(conn)
	if err != nil {
		fmt.Println("Error handling connection: ", err.Error())
	}
}

// HTTP Router
// ---

type HTTPRouter struct {
	routes []HTTPRoute
}

func (router *HTTPRouter) RegisterRoute(route HTTPRoute) {
	router.routes = append(router.routes, route)
}

func (router *HTTPRouter) GET(path string, handle func(Request) (Response, error)) {
	router.RegisterRoute(HTTPRoute{
		Verb:   "GET",
		Path:   path,
		Handle: handle,
	})
}

func (router *HTTPRouter) POST(path string, handle func(Request) (Response, error)) {
	router.RegisterRoute(HTTPRoute{
		Verb:   "POST",
		Path:   path,
		Handle: handle,
	})
}

func (router HTTPRouter) matchingRoute(request Request) *HTTPRoute {
	for _, route := range router.routes {
		if route.Verb == request.Verb {
			path.Match(route.Path, request.Path)
			return &route
		}
	}
	return nil
}

type HTTPRoute struct {
	Verb   string
	Path   string
	Handle func(Request) (Response, error)
}

func (router *HTTPRouter) processConnection(conn net.Conn) error {
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

	route := router.matchingRoute(unwrapedRequest)
	if route == nil {
		Response{StatusCode: NotFound}.write(conn)
	}
	response, err := route.Handle(unwrapedRequest)
	if err != nil {
		return err
	}

	// Write response
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
