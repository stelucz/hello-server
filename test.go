package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServerAddr = "localhost:8989"
	defaultSendPeriod = 5 * time.Second
	serverListenAddr  = ":8989"

	defaultLoggenPeriod    = 1 * time.Second
	defaultLoggenMsgLength = 100
	defaultLoggenMsgCount  = 1
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789     ") // Added more spaces
var logLevels = []string{"INFO", "WARNING", "ERROR", "DEBUG"}

func generateRandomString(length int) string {
    b := make([]rune, length)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    // Generate a random IP address
    ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
    return fmt.Sprintf("LOGGEN - %s - %s", ip, string(b))
}


func getRandomLogLevel() string {
	return logLevels[rand.Intn(len(logLevels))]
}

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

func runLogGenerator() {
	loggenPeriodStr := os.Getenv("LOGGEN_PERIOD")
	loggenPeriod := defaultLoggenPeriod
	if loggenPeriodStr != "" {
		parsedPeriod, err := time.ParseDuration(loggenPeriodStr)
		if err != nil {
			log.Printf("Invalid LOGGEN_PERIOD '%s', using default %v. Error: %v", loggenPeriodStr, defaultLoggenPeriod, err)
		} else {
			loggenPeriod = parsedPeriod
		}
	}

	loggenMsgLengthStr := os.Getenv("LOGGEN_MSG_LENGTH")
	loggenMsgLength := defaultLoggenMsgLength
	if loggenMsgLengthStr != "" {
		parsedLength, err := strconv.Atoi(loggenMsgLengthStr)
		if err != nil {
			log.Printf("Invalid LOGGEN_MSG_LENGTH '%s', using default %d. Error: %v", loggenMsgLengthStr, defaultLoggenMsgLength, err)
		} else {
			loggenMsgLength = parsedLength
		}
	}

	loggenMsgCountStr := os.Getenv("LOGGEN_MSG_COUNT")
	loggenMsgCount := defaultLoggenMsgCount
	if loggenMsgCountStr != "" {
		parsedCount, err := strconv.Atoi(loggenMsgCountStr)
		if err != nil {
			log.Printf("Invalid LOGGEN_MSG_COUNT '%s', using default %d. Error: %v", loggenMsgCountStr, defaultLoggenMsgCount, err)
		} else {
			loggenMsgCount = parsedCount
		}
	}

	log.Printf("Log generator configured to print %d message(s) of length %d every %v", loggenMsgCount, loggenMsgLength, loggenPeriod)

	ticker := time.NewTicker(loggenPeriod)
	defer ticker.Stop()

	for range ticker.C {
		for i := 0; i < loggenMsgCount; i++ {
			level := getRandomLogLevel()
			message := generateRandomString(loggenMsgLength)
			log.Printf("[%s] %s", level, message)
		}
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

	var conn net.Conn
	var err error
	for {
		conn, err = net.DialTimeout("tcp", serverAddr, 5*time.Second)
		if err == nil {
			log.Printf("Connected to server: %s", serverAddr)
			break // Successfully connected
		}
		log.Printf("Failed to connect to server %s: %v. Retrying in 5 seconds...", serverAddr, err)
		time.Sleep(5 * time.Second)
	}
	defer conn.Close()

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
	rand.Seed(time.Now().UnixNano()) // Seed random number generator

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <server|client|loggen>")
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "server":
		runServer()
	case "client":
		runClient()
	case "loggen":
		runLogGenerator()
	default:
		fmt.Printf("Unknown mode: %s. Use 'server', 'client', or 'loggen'.\n", mode)
		os.Exit(1)
	}
}
