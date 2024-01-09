package main

import (
	"errors"
	"strings"
)

type Request struct {
	Verb    string
	Path    string
	Headers map[string]string
	Body    []byte
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
	request.Verb, request.Path, err = parseStatusLine(statusLine)
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

	if len(requestParts) > 1 {
		request.Body = []byte(requestParts[1])
	}

	return &request, nil
}

func parseStatusLine(statusLine string) (term string, path string, err error) {
	fields := strings.Fields(statusLine)
	if len(fields) < 2 {
		return "", "", errors.New("fields not found")
	}

	term = strings.TrimSpace(fields[0])
	path = strings.TrimSpace(fields[1])
	return term, path, nil
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
