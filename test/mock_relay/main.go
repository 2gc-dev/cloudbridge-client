package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const (
	MessageTypeHello        = "hello"
	MessageTypeAuth         = "auth"
	MessageTypeAuthResponse = "auth_response"
	MessageTypeTunnelInfo   = "tunnel_info"
	MessageTypeError        = "error"
)

type Message struct {
	Type      string                 `json:"type"`
	Token     string                 `json:"token,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Features  []string               `json:"features,omitempty"`
	Status    string                 `json:"status,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	TunnelInfo *TunnelInfo           `json:"tunnel_info,omitempty"`
	Error     *ErrorMessage          `json:"error,omitempty"`
	ClientInfo map[string]interface{} `json:"client_info,omitempty"`
}

type TunnelInfo struct {
	TunnelID   string            `json:"tunnel_id"`
	LocalPort  int               `json:"local_port"`
	RemoteHost string            `json:"remote_host"`
	RemotePort int               `json:"remote_port"`
	Options    map[string]string `json:"options,omitempty"`
}

type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <port>")
		os.Exit(1)
	}

	port := os.Args[1]
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("Mock relay server listening on port %s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send hello message first
	helloMsg := Message{
		Type:     MessageTypeHello,
		Version:  "1.0.0",
		Features: []string{"tls", "jwt", "tunneling"},
	}

	if err := writeMessage(writer, helloMsg); err != nil {
		log.Printf("Failed to send hello: %v", err)
		return
	}

	// Read auth message
	authMsg, err := readMessage(reader)
	if err != nil {
		log.Printf("Failed to read auth: %v", err)
		return
	}

	if authMsg.Type != MessageTypeAuth {
		log.Printf("Expected auth message, got: %s", authMsg.Type)
		return
	}

	// Validate token (simple check for demo)
	token := authMsg.Token
	if token == "" {
		writeError(writer, "INVALID_TOKEN", "Token is required")
		return
	}

	// Send auth response
	authResp := Message{
		Type:     MessageTypeAuthResponse,
		Status:   "ok",
		ClientID: "test-client-001",
	}

	if err := writeMessage(writer, authResp); err != nil {
		log.Printf("Failed to send auth response: %v", err)
		return
	}

	fmt.Printf("Client authenticated successfully: %s\n", authMsg.ClientInfo)

	// Keep connection alive for a while
	for {
		msg, err := readMessage(reader)
		if err != nil {
			log.Printf("Connection closed: %v", err)
			break
		}

		switch msg.Type {
		case MessageTypeTunnelInfo:
			handleTunnelInfoFlat(writer, msg)
		default:
			log.Printf("Unknown message type: %s, full message: %+v", msg.Type, msg)
		}
	}
}

func handleTunnelInfoFlat(writer *bufio.Writer, msg *Message) {
	log.Printf("Received tunnel info: %+v", msg)
	
	// Create response with tunnel_id in root
	tunnelResp := map[string]interface{}{
		"type":       "tunnel_response",
		"status":     "ok",
		"tunnel_id":  "tunnel_001",
		"local_port": 3389,
		"remote_host": "192.168.1.100",
		"remote_port": 3389,
		"protocol":   "tcp",
	}

	data, err := json.Marshal(tunnelResp)
	if err != nil {
		log.Printf("Failed to marshal tunnel response: %v", err)
		return
	}

	data = append(data, '\n')
	if _, err := writer.Write(data); err != nil {
		log.Printf("Failed to send tunnel response: %v", err)
		return
	}

	if err := writer.Flush(); err != nil {
		log.Printf("Failed to flush tunnel response: %v", err)
		return
	}

	fmt.Printf("Tunnel created: tunnel_001 -> 192.168.1.100:3389\n")
}

func readMessage(reader *bufio.Reader) (*Message, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line = strings.TrimSpace(line)
	var msg Message
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

func writeMessage(writer *bufio.Writer, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	if _, err := writer.Write(data); err != nil {
		return err
	}

	return writer.Flush()
}

func writeError(writer *bufio.Writer, code, message string) error {
	errorMsg := Message{
		Type: MessageTypeError,
		Error: &ErrorMessage{
			Code:    code,
			Message: message,
		},
	}

	return writeMessage(writer, errorMsg)
} 