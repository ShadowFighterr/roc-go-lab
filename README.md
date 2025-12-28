
```markdown
# RPC Client–Server System in Go (AWS EC2)

## Overview

This project implements a simple **Remote Procedure Call (RPC)** system using **Go** and **TCP sockets**.  
The system consists of two independent components:

- **RPC Server** – runs on one AWS EC2 instance and listens for incoming RPC requests.
- **RPC Client** – runs on a separate AWS EC2 instance and sends requests to the server.

The client and server communicate over the network using **JSON-encoded messages**.  
The system supports **timeouts, retries, and failure scenarios**, demonstrating real-world RPC behavior in a distributed environment.

---

## Architecture

```

+-------------------+        TCP (port 6000)        +-------------------+
|   RPC Client EC2  |  --------------------------> |   RPC Server EC2  |
|  (Ubuntu, Go)     |                               |  (Ubuntu, Go)     |
+-------------------+                               +-------------------+

```

- Each component runs on its own EC2 instance.
- The server listens on `0.0.0.0:6000`.
- AWS Security Groups allow inbound TCP traffic on port `6000` to the server.

---

## Features

- TCP-based RPC communication
- JSON request/response format
- Client-side **timeouts**
- Client-side **retries**
- Demonstration of **at-least-once RPC semantics**
- Failure handling:
  - Slow responses
  - Server crash

---

## Technologies Used

- Go (Golang)
- AWS EC2 (Ubuntu 24.04)
- TCP sockets
- JSON
- Linux command-line tools

---

## Project Structure

```

rpc-go-lab/
├── client.go        # RPC client implementation
├── server.go        # RPC server implementation
├── go.mod           # Go module definition
├── README.md        # Project documentation

````

---

## Deployment Environment

- **Operating System:** Ubuntu Server 24.04 LTS
- **Instance Type:** t3.micro
- **Cloud Provider:** AWS EC2
- **Network:** Public IPv4, Security Groups configured for TCP/6000

---

## How to Build

### On both server and client EC2 instances:

```bash
sudo apt update
sudo apt install -y golang-go build-essential
````

---

## Running the Server

On the **server EC2 instance**:

```bash
cd ~/rpc-go-lab
go build -o rpc-server server.go
nohup ./rpc-server -addr 0.0.0.0 -port 6000 > server.log 2>&1 &
```

Verify the server is running:

```bash
ss -lnt | grep 6000
```

Expected output:

```
LISTEN 0 4096 0.0.0.0:6000
```

---

## Running the Client

On the **client EC2 instance**:

```bash
cd ~/rpc-go-lab
go build -o rpc-client client.go
```

### Example: Successful RPC call

```bash
./rpc-client -server <SERVER_PUBLIC_IP>:6000 -method add -params '{"a":5,"b":7}' -timeout 3 -retries 3
```

Expected response:

```json
{
  "result": 12,
  "status": "OK"
}
```

---

## Failure Demonstrations

### 1. Timeout and Retries

```bash
./rpc-client -server <SERVER_PUBLIC_IP>:6000 -method slow -params '{"sleep":5}' -timeout 2 -retries 3
```

* Client retries the request.
* Server processes requests with delay.
* Demonstrates timeout and retry behavior.

---

### 2. Server Crash

```bash
./rpc-client -server <SERVER_PUBLIC_IP>:6000 -method crash -params '{}'
```

* Server process terminates.
* Client observes failure.
* Demonstrates lack of guaranteed exactly-once semantics.

---

## RPC Semantics

This system provides **at-least-once RPC semantics**:

* The client retries requests when timeouts occur.
* A request may be executed more than once.
* The system does not guarantee exactly-once execution.

This behavior is typical for basic RPC systems without distributed transaction support.

---

## Security Notes

* SSH access is restricted to the user’s IP.
* Server allows inbound TCP traffic only on port `6000`.
* Private SSH keys are stored securely with proper permissions.

---

## Conclusion

This project demonstrates a functional RPC system deployed on AWS EC2 using Go.
It highlights core distributed systems concepts including networking, failure handling, retries, and RPC semantics.

---

## Author

Azamat Ubaidullauly
