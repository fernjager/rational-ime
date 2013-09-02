package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	ZHUYIN_QUERY    int = 0
	PINYIN_QUERY    int = 1
	DEFINITON_QUERY int = 2
)

// Socket is a struct containing WebSocket a connection handle
type socket struct {
	io.ReadWriter
	done chan bool
}

// ServerParams is a struct that stores server configuration and handles
type ServerParams struct {
	ref *ReferenceStore
}

// Request is a struct that represents the JSON object that is expected
// to be received by the server as a request
type Request struct {
	SessionID string
	QueryType int
	Query     string
	Timestamp int64
}

// Response is a struct that represents the JSON object that is sent
// back to the client
type Response struct {
	SessionID    string
	ResponseType int
	Data         interface{}
	Timestamp    int64
}

// Closes the socket object
func (s socket) Close() error {
	s.done <- true
	return nil
}

// Handle all GET requests
func (serv *ServerParams) requestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	path := strings.Split(r.URL.Path[1:], "/")
	var returnValue *[]Character
	if len(path) < 3 {
		fmt.Fprintf(w, "{code:500}")
		return
	}
	switch path[1] {
	case "zhuyin":
		returnValue, _ = serv.ref.GetByZhuyin(path[2])
	case "pinyin":
		returnValue, _ = serv.ref.GetByPinyin(path[2])
	case "def":
		returnValue, _ = serv.ref.GetByDefinition(path[2])
	case "char":
		returnValue, _ = serv.ref.GetByChar(path[2])
	default:
		{
			fmt.Fprintf(w, "{code:500}")
			return
		}
	}

	bytearray, _ := json.Marshal(Response{"102", 0, returnValue, 0})
	fmt.Fprintf(w, string(bytearray))
}

// errorHandler prints out default error message for GET requests
func (serv *ServerParams) errorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{code:500}")
}

// socketHandler handles WebSocket connections
func (serv *ServerParams) socketHandler(ws *websocket.Conn) {
	s := socket{ws, make(chan bool)}
	var buffer string
	fmt.Fscan(s, &buffer)
	fmt.Println("Received:", buffer)
	fmt.Fprint(s, "How do you do?")
	<-s.done
}

func InitServer(ref *ReferenceStore) {
	serv := ServerParams{ref}

	// WebSocket connection handler
	http.Handle("/socket", websocket.Handler(serv.socketHandler))

	// Old Get request handler
	http.HandleFunc("/get/", serv.requestHandler)

	// Default not found handler
	http.HandleFunc("/", serv.errorHandler)

	// Listen and Serve
	http.ListenAndServe(":8081", nil)
}
