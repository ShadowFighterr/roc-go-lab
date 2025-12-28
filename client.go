package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Request struct {
	RequestID string                 `json:"request_id"`
	Method    string                 `json:"method"`
	Params    map[string]interface{} `json:"params"`
	Timestamp string                 `json:"timestamp,omitempty"`
}

type Response struct {
	RequestID string      `json:"request_id"`
	Result    interface{} `json:"result,omitempty"`
	Status    string      `json:"status"`
	Error     string      `json:"error,omitempty"`
}

func main() {
	server := flag.String("server", "", "server address host:port (required)")
	method := flag.String("method", "add", "method to call (add|get_time|reverse_string|slow|crash|echo)")
	params := flag.String("params", "{}", "json string of params, e.g. '{\"a\":5,\"b\":7}'")
	timeout := flag.Int("timeout", 2, "per-request timeout seconds")
	maxRetries := flag.Int("retries", 3, "max number of attempts")
	flag.Parse()

	if *server == "" {
		fmt.Fprintln(os.Stderr, "server flag is required")
		flag.Usage()
		os.Exit(1)
	}

	var paramMap map[string]interface{}
	if err := json.Unmarshal([]byte(*params), &paramMap); err != nil {
		log.Fatalf("invalid params json: %v", err)
	}

	reqID := genUUID()
	req := Request{
		RequestID: reqID,
		Method:    *method,
		Params:    paramMap,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	var lastErr error
	for attempt := 1; attempt <= *maxRetries; attempt++ {
		log.Printf("Attempt %d/%d for request %s", attempt, *maxRetries, reqID)
		resp, err := sendRequest(*server, &req, time.Duration(*timeout)*time.Second)
		if err == nil {
			// success
			j, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Printf("Response:\n%s\n", string(j))
			return
		}
		lastErr = err
		log.Printf("Attempt %d error: %v", attempt, err)
		// exponential backoff with jitter
		backoff := time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond
		jitter := time.Duration(randInt(0, 200)) * time.Millisecond
		time.Sleep(backoff + jitter)
	}
	log.Fatalf("All attempts failed. last error: %v", lastErr)
}

func sendRequest(server string, req *Request, timeout time.Duration) (*Response, error) {
	// Dial with timeout
	conn, err := net.DialTimeout("tcp", server, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial error: %w", err)
	}
	defer conn.Close()

	// set deadline for read+write
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		return nil, fmt.Errorf("encode/send: %w", err)
	}

	dec := json.NewDecoder(conn)
	var resp Response
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode/receive: %w", err)
	}

	// ensure request_id matches
	if strings.TrimSpace(resp.RequestID) != req.RequestID {
		return nil, fmt.Errorf("mismatched request id in response: got %s expected %s", resp.RequestID, req.RequestID)
	}

	if resp.Status != "OK" {
		return &resp, fmt.Errorf("server error: %s", resp.Error)
	}
	return &resp, nil
}

// genUUID returns a v4-style random id string
func genUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	// set version to 4
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// small random int for jitter
func randInt(min, max int) int {
	if max <= min {
		return min
	}
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	// scale byte to range
	return min + int(b[0])%(max-min+1)
}
