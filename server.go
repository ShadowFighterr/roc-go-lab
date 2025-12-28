package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Message types
type Request struct {
	RequestID string                 `json:"request_id"`
	Method    string                 `json:"method"`
	Params    map[string]interface{} `json:"params"`
	Timestamp string                 `json:"timestamp,omitempty"`
}

type Response struct {
	RequestID string      `json:"request_id"`
	Result    interface{} `json:"result,omitempty"`
	Status    string      `json:"status"` // "OK" or "ERROR"
	Error     string      `json:"error,omitempty"`
}

func main() {
	port := flag.Int("port", 5000, "port to listen on")
	addr := flag.String("addr", "0.0.0.0", "address to bind")
	flag.Parse()

	listenAddr := fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("Starting RPC server on %s", listenAddr)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	dec := json.NewDecoder(conn)
	var req Request
	if err := dec.Decode(&req); err != nil {
		log.Printf("[%s] decode error: %v", remote, err)
		sendError(conn, "", "invalid json")
		return
	}

	log.Printf("[%s] Received request id=%s method=%s params=%v", remote, req.RequestID, req.Method, req.Params)
	resp := processRequest(&req)
	enc := json.NewEncoder(conn)

	// simulate a situation where server might crash after processing but before sending:
	if strings.ToLower(req.Method) == "crash" {
		log.Printf("Crash requested by client. Exiting server process.")
		// Send response before crash to show partial scenarios, optionally:
		_ = enc.Encode(resp) // ignore error
		// exit immediately (simulate crash)
		os.Exit(1)
	}

	if err := enc.Encode(resp); err != nil {
		log.Printf("[%s] encode error: %v", remote, err)
		return
	}
	log.Printf("[%s] Responded request id=%s status=%s", remote, req.RequestID, resp.Status)
}

func processRequest(req *Request) *Response {
	r := &Response{RequestID: req.RequestID}

	switch strings.ToLower(req.Method) {
	case "add":
		a, b, err := getTwoInts(req.Params, "a", "b")
		if err != nil {
			r.Status = "ERROR"
			r.Error = err.Error()
			return r
		}
		r.Result = a + b
		r.Status = "OK"
	case "reverse_string":
		sv, ok := req.Params["s"]
		if !ok {
			r.Status = "ERROR"
			r.Error = "missing param 's'"
			return r
		}
		s, ok := sv.(string)
		if !ok {
			r.Status = "ERROR"
			r.Error = "param 's' must be string"
			return r
		}
		r.Result = reverseString(s)
		r.Status = "OK"
	case "get_time":
		r.Result = time.Now().Format(time.RFC3339)
		r.Status = "OK"
	case "slow":
		// optional 'sleep' param: seconds to sleep
		secs := 5
		if sv, ok := req.Params["sleep"]; ok {
			switch t := sv.(type) {
			case float64:
				secs = int(t)
			case string:
				if v, err := strconv.Atoi(t); err == nil {
					secs = v
				}
			}
		}
		log.Printf("Simulating slow processing: sleeping %d seconds", secs)
		time.Sleep(time.Duration(secs) * time.Second)
		r.Result = fmt.Sprintf("slept %d seconds", secs)
		r.Status = "OK"
	case "echo":
		r.Result = req.Params
		r.Status = "OK"
	default:
		r.Status = "ERROR"
		r.Error = fmt.Sprintf("unknown method '%s'", req.Method)
	}
	return r
}

func getTwoInts(params map[string]interface{}, ka, kb string) (int, int, error) {
	av, ok := params[ka]
	if !ok {
		return 0, 0, fmt.Errorf("missing param '%s'", ka)
	}
	bv, ok := params[kb]
	if !ok {
		return 0, 0, fmt.Errorf("missing param '%s'", kb)
	}
	a, err := asInt(av)
	if err != nil {
		return 0, 0, fmt.Errorf("param '%s' error: %v", ka, err)
	}
	b, err := asInt(bv)
	if err != nil {
		return 0, 0, fmt.Errorf("param '%s' error: %v", kb, err)
	}
	return a, b, nil
}

func asInt(v interface{}) (int, error) {
	switch t := v.(type) {
	case float64:
		return int(t), nil
	case int:
		return t, nil
	case string:
		if iv, err := strconv.Atoi(t); err == nil {
			return iv, nil
		}
	}
	return 0, errors.New("not an integer")
}

func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// sendError sends a simple error response with optional requestID
func sendError(conn net.Conn, reqID string, msg string) {
	resp := Response{
		RequestID: reqID,
		Status:    "ERROR",
		Error:     msg,
	}
	_ = json.NewEncoder(conn).Encode(resp)
}

// small helper to produce a short request id for server logs (not used in server main flow)
func newShortID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
