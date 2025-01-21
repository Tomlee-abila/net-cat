
# Net Cat - TCP Chat Project



## Overview

This project is a recreation of the NetCat (nc) command-line utility in a server-client architecture. The implementation supports server mode to listen for incoming connections on a specified port and client mode to connect to a server, enabling group chat functionality.

### Features
1. **TCP Server-Client Connection**: Supports multiple clients connecting to a server via TCP.
2. **Named Clients**: Each client is required to provide a name before joining the chat.
3. **Group Chat**: Allows clients to exchange messages in a shared chat room.
4. **Message Identification**: Messages include a timestamp and the sender's name in the format:  
   `[YYYY-MM-DD HH:MM:SS][client.name]:[message]`.
5. **Message History**: New clients receive the complete message history upon joining.
6. **Connection Notifications**: 
   - All clients are notified when a new client joins.
   - Clients are informed when someone leaves the chat.
7. **Connection Control**: Maximum of 10 simultaneous connections.
8. **Error Handling**: Manages errors gracefully on both server and client sides.
9. **Default Port**: If no port is specified, the server listens on port `8989` by default.
10. **Empty Messages**: Empty messages are not broadcasted.

## Setup

1. Clone the repository:
```
git https://learn.zone01kisumu.ke/git/tabila/net-cat.git
cd net-cat
```

2. Build the project:
```
go build -o TCPChat main.go 
```


## Usage

### Start the Server
```bash
./TCPChat
```

### Connect a Client
Use `nc` to connect to the server:
```bash
$ nc <IP> <PORT>
```

### Example Interaction

#### Client 1
```bash
$ nc localhost 2525
Welcome to TCP-Chat!
         _nnnn_
        dGGGGMMb
       @p~qp~~qMb
       M|@||@) M|
       @,----.JM|
      JS^\__/  qKL
     dZP        qKRb
    dZP          qKKb
   fZP            SMMb
   HZM            MMMM
   FqM            MMMM
 __| ".        |\dS"qML
 |    `.       | `' \Zq
_)      \.___.,|     .'
\____   )MMMMMP|   .'
     `-'       `--'
[ENTER YOUR NAME]: Alice
[2025-01-20 12:30:00][Alice]:Hello, everyone!
```

#### Client 2
```bash
$ nc localhost 2525
Welcome to TCP-Chat!
         _nnnn_
        dGGGGMMb
       @p~qp~~qMb
       M|@||@) M|
       @,----.JM|
      JS^\__/  qKL
     dZP        qKRb
    dZP          qKKb
   fZP            SMMb
   HZM            MMMM
   FqM            MMMM
 __| ".        |\dS"qML
 |    `.       | `' \Zq
_)      \.___.,|     .'
\____   )MMMMMP|   .'
     `-'       `--'
[ENTER YOUR NAME]: Bob
[2025-01-20 12:30:10][Bob]:Hi Alice!
```

### Error Handling
- If a port is not provided:
  ```bash
  $ go run . localhost
  [USAGE]: ./TCPChat $port
  ```

### Example Logs
```plaintext
[2025-01-20 12:30:00][Alice]:Hello, everyone!
Bob has joined the chat.
[2025-01-20 12:30:10][Bob]:Hi Alice!
Alice has left the chat.
[2025-01-20 12:35:00][Bob]:Goodbye!
```

## Contribution  
This project was collaboratively developed by **Tabila**, **Kevwasonga**, and **Aadero**.