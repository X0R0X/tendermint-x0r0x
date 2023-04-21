package v0

import (
	"fmt"
	"github.com/tendermint/tendermint/types"
	"net"
)

type ClientManager struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	socket net.Conn
	data   chan []byte
}

type JsonMempoolTxMsg struct {
	Type      byte   `json:"type"`
	Height    int64  `json:"height"`
	GasWanted int64  `json:"gasWanted"`
	Tx        []byte `json:"tx"`
}

type JsonNewBlockMsg struct {
	Type   byte      `json:"type"`
	Height int64     `json:"height"`
	Txs    types.Txs `json:"txs"`
}

var manager ClientManager

// ReportTxs Report Transactions as soon as they're included in the mempool
const ReportTxs bool = true

// ReportBlocks Report New Blocks as soon as our node is notified
const ReportBlocks bool = true

// ReportIncludeTxsInBlock Include Raw Txs that were included in the block
const ReportIncludeTxsInBlock bool = false

func (manager *ClientManager) start() {
	for {
		select {
		case connection := <-manager.register:
			manager.clients[connection] = true
			fmt.Println("Added new connection!")

		case connection := <-manager.unregister:
			if manager.clients[connection] {
				close(connection.data)
				delete(manager.clients, connection)
				fmt.Println("A connection has terminated!")
			}

		case message := <-manager.broadcast:
			for connection := range manager.clients {
				select {
				case connection.data <- message:
				default:
					close(connection.data)
					delete(manager.clients, connection)
				}
			}
		}
	}
}

func (manager *ClientManager) receive(client *Client) {
	for {
		message := make([]byte, 4096)
		length, err := client.socket.Read(message)
		if err != nil {
			manager.unregister <- client
			err = client.socket.Close()
			if err != nil {
				fmt.Println(err)
			}
			break
		}
		if length > 0 {
			fmt.Println("RECEIVED: " + string(message))
			manager.broadcast <- message
		}
	}
}

func (client *Client) receive() {
	for {
		message := make([]byte, 4096)
		length, err := client.socket.Read(message)
		if err != nil {
			err = client.socket.Close()
			if err != nil {
				fmt.Println(err)
			}
			break
		}
		if length > 0 {
			fmt.Println("RECEIVED: " + string(message))
		}
	}
}

func (manager *ClientManager) send(client *Client) {
	defer func(socket net.Conn) {
		err := socket.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(client.socket)
	for {
		select {
		case message, ok := <-client.data:
			if !ok {
				manager.unregister <- client
				return
			}
			_, err := client.socket.Write(message)
			if err != nil {
				manager.unregister <- client
			}
		}
	}
}

func startServer() {
	fmt.Println("Starting server...")
	listener, err := net.Listen("tcp", ":6666")
	if err != nil {
		fmt.Println(err)
		return
	}
	manager = ClientManager{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go manager.start()
	for {
		connection, _ := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		client := &Client{socket: connection, data: make(chan []byte)}
		manager.register <- client
		//go manager.receive(client)
		go manager.send(client)
	}
}

func SnStartServer() {
	fmt.Println("Starting server...")

	listener, err := net.Listen("tcp", ":6666")
	if err != nil {
		fmt.Println(err)
		return
	}
	manager = ClientManager{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go manager.start()
	for {
		connection, _ := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		client := &Client{socket: connection, data: make(chan []byte)}
		manager.register <- client
		//go manager.receive(client)
		go manager.send(client)
	}
}

func SnBroadcast(msg []byte) {
	for clientPtr := range manager.clients {
		clientPtr.data <- msg
	}
}
