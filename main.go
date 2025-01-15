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
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	clients    []Client
	messages   string
}

func (s *Server) addClient(Client Client) {
	s.clients = append(s.clients, Client)
}

func (s *Server) removeClient(client Client) {
	for i, c := range s.clients {
		if c.ipAdd == client.ipAdd {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
		}
	}
}

func (s *Server) messageClients(client Client, message string, tf string) {
	s.messages += message
	for _, c := range s.clients {
		if c.ipAdd != client.ipAdd {
			c.conn.Write([]byte(message))
			c.conn.Write([]byte("\n" + tf + "[" + c.name + "]:"))
		}
	}
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		messages:   "",
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
	// close(s.msgch)
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
		// buf := make([]byte, 2048)
		// n, err := conn.Read(buf)

		reader := bufio.NewReader(conn)
		Name, err := reader.ReadString('\n')

		// Name := string(buf[:n])
		Name = strings.Replace(Name, "\r", "", -1)
		Name = strings.Replace(Name, "\n", "", -1)
		// fmt.Println()
		// fmt.Print(Name[len(Name)-2])

		client := Client{name: Name, conn: conn, ipAdd: conn.RemoteAddr().String()}
		s.addClient(client)

		conn.Write([]byte(s.messages+"\n"))

		// notify all clients that there is a new client
		t := time.Now()
		tf := "[" + t.Format("02-01-2006 15:04:05") + "]"

		s.messageClients(client, "\n"+client.name+" has joined our chat...", tf)

		go s.readLoop(conn, client)
	}
}

func (s *Server) readLoop(conn net.Conn, client Client) {
	defer conn.Close()

	buf := make([]byte, 2048)

	for {
		t := time.Now()

		tf := "[" + t.Format("02-01-2006 15:04:05") + "]"

		conn.Write([]byte(tf + "[" + client.name + "]:"))
		n, err := conn.Read(buf)
		if err != nil {
			s.messageClients(client, "\n"+client.name+" has left our chat...", tf)
			s.removeClient(client)
			return
		}

		payload := string(buf[:n])
		payload = strings.Replace(payload, "\r", "", -1)
		payload = strings.Replace(payload, "\n", "", -1)

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
	}
	port := "8989"

	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	server := NewServer(":" + port)

	if err := server.Start(); err != nil {
		// fmt.Println("err:", err)
		port = "8989"
		server = NewServer(":" + port)
		log.Fatal(server.Start())
	}
	fmt.Printf("Listening on the port :%s\n", port)
}
