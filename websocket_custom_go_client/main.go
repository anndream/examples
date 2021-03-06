package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/kataras/iris.v6"

	"golang.org/x/net/websocket"
)

// WS is the current websocket connection
var WS *websocket.Conn

// Note: the original example(use old code) is not maden by me, but it was a post issue by a user to fix a bug
// see more here: https://github.com/kataras/go-websocket/issues/24
func main() {
	if len(os.Args) == 2 && strings.ToLower(os.Args[1]) == "server" {
		ServerLoop()
	} else if len(os.Args) == 2 && strings.ToLower(os.Args[1]) == "client" {
		ClientLoop()
	} else {
		fmt.Println("wsserver [server|client]")
	}
}

/////////////////////////////////////////////////////////////////////////
// client side
func sendUntilErr(sendInterval int) {
	i := 1
	for {
		time.Sleep(time.Duration(sendInterval) * time.Second)
		err := SendMessage("2", "all", "objectupdate", "2.UsrSchedule_v1_1")
		if err != nil {
			fmt.Println("failed to send join message", err.Error())
			return
		}
		fmt.Println("objectupdate", i)
		i++
	}
}

func recvUntilErr() {
	var msg = make([]byte, 2048)
	var n int
	var err error
	i := 1
	for {
		if n, err = WS.Read(msg); err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("%v Received: %s.%v\n", time.Now(), string(msg[:n]), i)
		i++
	}

}

//ConnectWebSocket connect a websocket to host
func ConnectWebSocket() error {
	var origin = "http://localhost/"
	var url = "ws://localhost:9090/socket"
	var err error
	WS, err = websocket.Dial(url, "", origin)
	return err
}

// CloseWebSocket closes the current websocket connection
func CloseWebSocket() error {
	if WS != nil {
		return WS.Close()
	}
	return nil
}

// SendMessage broadcast a message to server
func SendMessage(serverID, to, method, message string) error {
	buffer := []byte(message)
	return SendtBytes(serverID, to, method, buffer)
}

// SendtBytes broadcast a message to server
func SendtBytes(serverID, to, method string, message []byte) error {
	buffer := []byte(fmt.Sprintf("go-websocket-message:%v;0;%v;%v;", method, serverID, to))
	buffer = append(buffer, message...)
	_, err := WS.Write(buffer)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// ClientLoop connects to websocket server, the keep send and recv dataS
func ClientLoop() {
	for {
		time.Sleep(time.Second)
		err := ConnectWebSocket()
		if err != nil {
			fmt.Println("failed to connect websocket", err.Error())
			continue
		}
		// time.Sleep(time.Second)
		err = SendMessage("2", "all", "join", "dummy2")
		go sendUntilErr(2)
		recvUntilErr()
		err = CloseWebSocket()
		if err != nil {
			fmt.Println("failed to close websocket", err.Error())
		}
	}

}

/////////////////////////////////////////////////////////////////////////
// server side

// OnConnect handles incoming websocket connection
func OnConnect(c iris.WebsocketConnection) {
	fmt.Println("socket.OnConnect()")
	c.On("join", func(message string) { OnJoin(message, c) })
	c.On("objectupdate", func(message string) { OnObjectUpdated(message, c) })
	// ok works too c.EmitMessage([]byte("dsadsa"))
	c.OnDisconnect(func() { OnDisconnect(c) })

}

// ServerLoop listen and serve websocket requests
func ServerLoop() {
	// // the path which the websocket client should listen/registed to ->
	iris.Config.Websocket.Endpoint = "/socket"
	iris.Websocket.OnConnection(OnConnect)
	iris.Listen("0.0.0.0:9090")

}

// OnJoin handles Join broadcast group request
func OnJoin(message string, c iris.WebsocketConnection) {
	t := time.Now()
	c.Join("server2")
	fmt.Println("OnJoin() time taken:", time.Since(t))
}

// OnObjectUpdated broadcasts to all client an incoming message
func OnObjectUpdated(message string, c iris.WebsocketConnection) {
	t := time.Now()
	s := strings.Split(message, ";")
	if len(s) != 3 {
		fmt.Println("OnObjectUpdated() invalid message format:" + message)
		return
	}
	serverID, _, objectID := s[0], s[1], s[2]
	err := c.To("server"+serverID).Emit("objectupdate", objectID)
	if err != nil {
		fmt.Println(err, "failed to broacast object")
		return
	}
	fmt.Println(fmt.Sprintf("OnObjectUpdated() message:%v, time taken: %v", message, time.Since(t)))
}

// OnDisconnect clean up things when a client is disconnected
func OnDisconnect(c iris.WebsocketConnection) {
	c.Leave("server2")
	fmt.Println("OnDisconnect(): client disconnected!")

}
