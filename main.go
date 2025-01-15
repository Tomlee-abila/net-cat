package main

import (
	"fmt"
	"log"
	"net"
	"os"
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
	msgch      chan Message
	clients    []Client
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
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
	close(s.msgch)
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
		buf := make([]byte, 2048)
		n, err := conn.Read(buf)

		name := string(buf[:n-1])

		fmt.Println("New connection to the Server:", name, conn.RemoteAddr())
		go s.readLoop(conn, name)
	}
}

func (s *Server) readLoop(conn net.Conn, name string) {
	defer conn.Close()

	buf := make([]byte, 2048)

	for {
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Lee has left our chat...")
			return
		}

		s.msgch <- Message{
			from:    conn.RemoteAddr().String(),
			payload: buf,
			name:    name,
		}

		// conn.Write([]byte("Thank you for your message!"))

		// msg := buf[:n]
		// fmt.Println(string(msg))

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

	go func() {
		for msg := range server.msgch {
			t := time.Now()
			fmt.Printf("[%s][%s]:%s", t.Format("02-01-2006 15:04:05"), msg.name, string(msg.payload))
		}
	}()
	if err := server.Start(); err != nil {
		// fmt.Println("err:", err)
		port = "8989"
		server = NewServer(":" + port)
		log.Fatal(server.Start())
	}
	fmt.Printf("Listening on the port :%s\n", port)
}
