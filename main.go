package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Message struct {
	from    string
	name    string
	payload []byte
}

type Client struct {
	conn  net.Conn
	ipAdd string
	name  string
}

type Server struct {
	listenAddr    string
	ln           net.Listener
	quitch       chan struct{}
	clients      []Client
	messages     string
	maxClients   int
	mu           sync.Mutex // Mutex for thread-safe operations
	activeNames  map[string]bool
}

func (s *Server) addClient(Client Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.clients) >= s.maxClients {
		return errors.New("server is full")
	}

	if s.activeNames[Client.name] {
		return errors.New("username already taken")
	}

	s.clients = append(s.clients, Client)
	s.activeNames[Client.name] = true
	return nil
}

func (s *Server) removeClient(client Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.clients {
		if c.ipAdd == client.ipAdd {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			delete(s.activeNames, client.name)
			break
		}
	}
}

func validateUsername(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("username cannot be empty")
	}
	if len(name) > 32 {
		return errors.New("username must be 32 characters or less")
	}
	if strings.ContainsAny(name, "\n\r\t") {
		return errors.New("username contains invalid characters")
	}
	return nil
}

func (s *Server) messageClients(client Client, message string, tf string) {
	s.mu.Lock()
	s.messages += message
	s.mu.Unlock()

	for _, c := range s.clients {
		if c.ipAdd != client.ipAdd {
			c.conn.Write([]byte(message))
			c.conn.Write([]byte("\n" + tf + "[" + c.name + "]:"))
		}
	}

	// Create or open the log file
	logFile, err := os.OpenFile("server_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()

	// Create a bufio.Writer
	writer := bufio.NewWriter(logFile)

	// Write the message to the log file
	_, err = writer.WriteString(message)
	if err != nil {
		fmt.Println("Error writing to log file:", err)
		return
	}

	// Flush the writer to ensure the data is written to the file immediately
	err = writer.Flush()
	if err != nil {
		fmt.Println("Error flushing writer:", err)
	}
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr:  listenAddr,
		quitch:      make(chan struct{}),
		messages:    "",
		maxClients:  10,
		activeNames: make(map[string]bool),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}

	defer ln.Close()

	go s.acceptLoop()

	s.ln = ln

	<-s.quitch
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept err:", err)
			continue
		}

		conn.Write([]byte("Welcome to TCP-Chat!\n         _nnnn_\n        dGGGGMMb\n       @p~qp~~qMb\n       M|@||@) M|\n       @,----.JM|\n      JS^\\__/  qKL\n     dZP        qKRb\n    dZP          qKKb\n   fZP            SMMb\n   HZM            MMMM\n   FqM            MMMM\n __| \".        |\\dS\"qML\n |    `.       | `' \\Zq\n_)      \\.___.,|     .'\n\\____   )MMMMMP|   .'\n     `-'       `--'\n[ENTER YOUR NAME]:"))

		reader := bufio.NewReader(conn)
		name, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading name:", err)
			conn.Close()
			continue
		}

		name = strings.Replace(name, "\r", "", -1)
		name = strings.Replace(name, "\n", "", -1)

		if err := validateUsername(name); err != nil {
			conn.Write([]byte(err.Error() + "\n"))
			conn.Close()
			continue
		}

		s.mu.Lock()
		if len(s.clients) >= s.maxClients {
			s.mu.Unlock()
			conn.Write([]byte("Server is full. Please try again later.\n"))
			conn.Close()
			continue
		}

		if s.activeNames[name] {
			s.mu.Unlock()
			conn.Write([]byte("Username already taken. Please choose another name.\n"))
			conn.Close()
			continue
		}
		s.mu.Unlock()

		client := Client{name: name, conn: conn, ipAdd: conn.RemoteAddr().String()}
		if err := s.addClient(client); err != nil {
			conn.Write([]byte(err.Error() + "\n"))
			conn.Close()
			continue
		}

		conn.Write([]byte(s.messages + "\n"))

		// notify all clients that there is a new client
		t := time.Now()
		tf := "[" + t.Format("02-01-2006 15:04:05") + "]"

		s.messageClients(client, "\n"+client.name+" has joined our chat...", tf)

		go s.readLoop(conn, client)
	}
}

func (s *Server) readLoop(conn net.Conn, client Client) {
	defer func() {
		conn.Close()
		s.removeClient(client)
	}()

	buf := make([]byte, 2048)

	for {
		t := time.Now()
		tf := "[" + t.Format("02-01-2006 15:04:05") + "]"

		conn.Write([]byte(tf + "[" + client.name + "]:"))
		n, err := conn.Read(buf)
		if err != nil {
			s.messageClients(client, "\n"+client.name+" has left our chat...", tf)
			return
		}

		payload := string(buf[:n])
		payload = strings.Replace(payload, "\r", "", -1)
		payload = strings.Replace(payload, "\n", "", -1)

		// Validate message length
		if len(payload) > 1024 {
			conn.Write([]byte("Message too long. Maximum length is 1024 characters.\n"))
			continue
		}

		message := "\n" + tf + "[" + client.name + "]:" + payload
		fmt.Print(message)

		if len(payload) > 1 {
			s.messageClients(client, message, tf)
		}
	}
}

func main() {
	if len(os.Args) > 2 {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}
	port := "8989"

	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	server := NewServer(":" + port)

	if err := server.Start(); err != nil {
		port = "8989"
		server = NewServer(":" + port)
		log.Fatal(server.Start())
	}
	fmt.Printf("Listening on the port :%s\n", port)
}
