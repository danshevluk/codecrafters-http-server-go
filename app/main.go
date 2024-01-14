package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var storageDirectory *string

func main() {
	storageDirectory = flag.String("directory", "storage", "Storage directory")
	flag.Parse()

	server := HTTPServer{
		Host: "0.0.0.0",
		Port: 4221,
	}
	server.Router = makeRouter()

	err := server.Serve()
	if err != nil {
		fmt.Println("Error serving: ", err.Error())
		os.Exit(1)
	}
}

func makeRouter() HTTPRouter {
	return HTTPRouter{
		routes: []HTTPRoute{
			{
				Verb:    "GET",
				Path:    "/",
				Handler: rootHandler,
			},
			{
				Verb:    "GET",
				Path:    "/echo",
				Handler: echoHandler,
			},
			{
				Verb:    "GET",
				Path:    "/user-agent",
				Handler: userAgentHandler,
			},
			{
				Verb:    "GET",
				Path:    "/files",
				Handler: getFilesHandler,
			},
			{
				Verb:    "POST",
				Path:    "/files",
				Handler: postFilesHandler,
			},
		},
	}
}

func rootHandler(request Request) (Response, error) {
	return Response{
		StatusCode: OK,
	}, nil
}

func echoHandler(request Request) (Response, error) {
	pathComponents := request.GetPathComponents()
	if len(pathComponents) < 2 {
		return Response{StatusCode: BadRequest}, nil
	}
	return Response{
		StatusCode: OK,
	}.withStringBody(strings.Join(pathComponents[1:], "/"), "text/plain"), nil
}

func userAgentHandler(request Request) (Response, error) {
	if request.Headers == nil || request.Headers["User-Agent"] == "" {
		return Response{StatusCode: BadRequest}, nil
	}

	userAgent := request.Headers["User-Agent"]
	return Response{
		StatusCode: OK,
	}.withStringBody(userAgent, "text/plain"), nil
}

func postFilesHandler(request Request) (Response, error) {
	pathComponents := request.GetPathComponents()
	if len(pathComponents) < 2 {
		return Response{StatusCode: BadRequest}, nil
	}

	filePath := strings.Join(pathComponents[1:], "/")
	directory := *storageDirectory
	path := filepath.Join(directory, filePath)

	// Read file contents from []byte and remove empty bytes
	fileContents := request.Body
	fileContents = bytes.Trim(fileContents, "\x00")

	os.WriteFile(path, fileContents, 0644)
	return Response{StatusCode: Created}, nil
}

func getFilesHandler(request Request) (Response, error) {
	pathComponents := request.GetPathComponents()
	if len(pathComponents) < 2 {
		return Response{StatusCode: BadRequest}, nil
	}

	filePath := strings.Join(pathComponents[1:], "/")
	directory := *storageDirectory
	path := filepath.Join(directory, filePath)

	// Reading file
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Response{StatusCode: NotFound}, nil
	}

	fileContents, err := os.ReadFile(path)
	if err != nil {
		return Response{StatusCode: ServerErr}, nil
	}

	return Response{
		StatusCode: OK,
	}.withBody(fileContents, "application/octet-stream"), nil
}
