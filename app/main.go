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

func makeRouter() HttpRouter {
	return HttpRouter{
		Routes: []HttpRoute{},
	}
}
