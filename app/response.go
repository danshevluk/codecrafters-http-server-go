package main

import (
	"fmt"
	"net"
)

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

const (
	OK         = 200
	Created    = 201
	BadRequest = 400
	NotFound   = 404
	ServerErr  = 500
)

func (r Response) statusText() string {
	switch r.StatusCode {
	case OK:
		return "OK"
	case Created:
		return "Created"
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
