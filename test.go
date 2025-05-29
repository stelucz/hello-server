package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	defaultServerAddr = "localhost:8989"
	defaultSendPeriod = 5 * time.Second
	serverListenAddr  = ":8989"
)

func runServer() {
	listener, err := net.Listen("tcp", serverListenAddr)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	log.Printf("Server listening on %s", serverListenAddr)

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
	clientAddr := conn.RemoteAddr().String()
	log.Printf("Client connected: %s", clientAddr)

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() != "EOF" { // EOF is expected when client disconnects
				log.Printf("Error reading from client %s: %v", clientAddr, err)
			} else {
				log.Printf("Client %s disconnected.", clientAddr)
			}
			return
		}
		log.Printf("Received from %s: %s", clientAddr, strings.TrimSpace(message))
	}
}

func runClient() {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = defaultServerAddr
		log.Printf("SERVER_ADDR not set, using default: %s", serverAddr)
	}

	sendPeriodStr := os.Getenv("SEND_PERIOD")
	sendPeriod := defaultSendPeriod
	if sendPeriodStr != "" {
		parsedPeriod, err := time.ParseDuration(sendPeriodStr)
		if err != nil {
			log.Printf("Invalid SEND_PERIOD '%s', using default %v. Error: %v", sendPeriodStr, defaultSendPeriod, err)
		} else {
			sendPeriod = parsedPeriod
		}
	}
	log.Printf("Client configured to send to %s every %v", serverAddr, sendPeriod)

	conn, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to server %s: %v", serverAddr, err)
	}
	defer conn.Close()
	log.Printf("Connected to server: %s", serverAddr)

	ticker := time.NewTicker(sendPeriod)
	defer ticker.Stop()

	var messageCount int64 = 0 // Initialize message counter

	for {
		select {
		case <-ticker.C:
			messageCount++ // Increment for new conceptual message

			var currentMessageToSend string
			maxSendAttempts := 3 // Define max attempts for a single message

			for attempt := 1; attempt <= maxSendAttempts; attempt++ {
				timestamp := time.Now().Format(time.RFC3339Nano)
				if attempt == 1 {
					currentMessageToSend = fmt.Sprintf("Hello #%d from client at %s\n", messageCount, timestamp)
				} else {
					currentMessageToSend = fmt.Sprintf("Hello #%d (attempt %d) from client at %s\n", messageCount, attempt, timestamp)
				}

				_, err := conn.Write([]byte(currentMessageToSend))
				if err == nil {
					// Message sent successfully
					log.Printf("Sent to %s: %s", serverAddr, strings.TrimSpace(currentMessageToSend))
					break // Exit attempt loop, wait for next ticker
				}

				// If write failed
				log.Printf("Failed to send message #%d (attempt %d): %v", messageCount, attempt, err)

				if attempt == maxSendAttempts {
					log.Fatalf("Failed to send message #%d after %d attempts. Exiting.", messageCount, maxSendAttempts)
				}

				log.Println("Attempting to reconnect...")
				conn.Close() // Close the old/failed connection

				newConn, dialErr := net.DialTimeout("tcp", serverAddr, 5*time.Second)
				if dialErr != nil {
					log.Printf("Failed to reconnect on attempt %d for message #%d: %v.", attempt, messageCount, dialErr)
					// Pause briefly before next attempt to avoid tight loop on connection errors
					time.Sleep(time.Second)
					continue // Continue to next attempt, will try to connect again
				}
				conn = newConn // Update connection
				log.Println("Reconnected to server. Retrying send.")
				// Loop will continue to the next attempt with the new connection
			}
		}
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds) // Add timestamps to logs

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <server|client>")
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		fmt.Printf("Unknown mode: %s. Use 'server' or 'client'.\n", mode)
		os.Exit(1)
	}
}
