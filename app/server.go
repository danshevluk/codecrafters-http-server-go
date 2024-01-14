package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const newline = "\r\n"

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
		Verb:    "GET",
		Path:    path,
		Handler: handle,
	})
}

func (router *HTTPRouter) POST(path string, handle func(Request) (Response, error)) {
	router.RegisterRoute(HTTPRoute{
		Verb:    "POST",
		Path:    path,
		Handler: handle,
	})
}

func (router HTTPRouter) matchingRoute(request Request) *HTTPRoute {
	for _, route := range router.routes {
		if route.Verb == request.Verb {
			if route.Path == request.Path {
				return &route
			} else if route.Path != "/" && strings.HasPrefix(request.Path, route.Path) {
				return &route
			}
		}
	}
	return nil
}

type HTTPRoute struct {
	Verb    string
	Path    string
	Handler func(Request) (Response, error)
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

	var response Response

	route := router.matchingRoute(unwrapedRequest)
	if route == nil {
		response = Response{StatusCode: NotFound}
	} else {
		response, err = route.Handler(unwrapedRequest)
		if err != nil {
			return err
		}
	}

	// Write response
	err = response.write(conn)
	if err != nil {
		return err
	}

	// Connection handled successfully
	return nil
}
