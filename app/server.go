package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
)

type Request struct {
	method    string
	path      string
	userAgent string
	postData  []string
}

type Response struct {
	code        string
	contentType string
	content     string
}

func getRequest(conn net.Conn) Request {
	bytes := make([]byte, 4096)
	n, err := conn.Read(bytes)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		os.Exit(1)
	}

	request := string(bytes[:n])
	lines := strings.Split(request, "\n")

	return Request{
		method:    strings.Split(lines[0], " ")[0],
		path:      strings.Split(lines[0], " ")[1],
		userAgent: strings.TrimPrefix(lines[2], "User-Agent: "),
		postData:  strings.Split(lines[len(lines)-1], "&"),
	}
}

func sendResponse(conn net.Conn, resp Response) {
	switch resp.code {
	case "200":
		fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\n")
		if resp.contentType == "" {
			fmt.Fprintf(conn, "\r\n")
			return
		}
	case "404":
		fmt.Fprintf(conn, "HTTP/1.1 404 Not Found\r\n\r\n")
		return
	case "201":
		fmt.Fprintf(conn, "HTTP/1.1 201 Created\r\n\r\n")
		return
	}

	fmt.Fprintf(conn, "Content-Type: %v\r\n", resp.contentType)
	fmt.Fprintf(conn, "Content-Length: %v\r\n", len(strings.ReplaceAll(resp.content, "/", "")))
	fmt.Fprintf(conn, "\r\n")

	fmt.Fprintf(conn, "%v\r\n", resp.content)
}

func handleClient(conn net.Conn) {
	request := getRequest(conn)

	echoRegexp := "/echo/(.+)"
	echoRegex, err := regexp.Compile(echoRegexp)
	if err != nil {
		fmt.Println("Error compiling regex: ", err.Error())
		os.Exit(1)
	}

	filesRegexp := "/files/(.+)"
	filesRegex, err := regexp.Compile(filesRegexp)
	if err != nil {
		fmt.Println("Error compiling regex: ", err.Error())
		os.Exit(1)
	}

	switch {
	case request.path == "/":
		sendResponse(conn, Response{code: "200"})

	case request.path == "/user-agent":
		sendResponse(conn, Response{
			code:        "200",
			contentType: "text/plain",
			content:     request.userAgent,
		})

	case echoRegex.MatchString(request.path):
		matches := echoRegex.FindStringSubmatch(request.path)
		sendResponse(conn, Response{
			code:        "200",
			contentType: "text/plain",
			content:     matches[1],
		})

	case filesRegex.MatchString(request.path):
		fileName := filesRegex.FindStringSubmatch(request.path)[1]
		fileName = dirFlag + "/" + fileName
		if request.method == "GET" {
			data, err := os.ReadFile(fileName)
			if err != nil {
				sendResponse(conn, Response{code: "404"})
			} else {
				sendResponse(conn, Response{
					code:        "200",
					contentType: "application/octet-stream",
					content:     string(data),
				})
			}
		} else if request.method == "POST" {
			os.WriteFile(fileName, []byte(request.postData[0]), 0333)
			sendResponse(conn, Response{code: "201"})
		}

	default:
		sendResponse(conn, Response{code: "404"})
	}

	conn.Close()
}

var dirFlag string

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	flag.StringVar(&dirFlag, "directory", "", "Where to save the files")
	flag.Parse()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleClient(conn)
	}
}
