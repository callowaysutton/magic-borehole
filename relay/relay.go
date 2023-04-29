package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"log"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Channel struct {
	senders   map[*websocket.Conn]struct{}
	receivers map[*websocket.Conn]struct{}
}

var channels = make(map[string]*Channel)
var channelsMutex sync.Mutex

var upgrader = websocket.Upgrader{}

var totalMessages = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "total_messages",
	Help: "The total number of messages relayed",
})

var totalReceived = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "total_received",
	Help: "The total number of messages received",
})

var totalSent = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "total_sent",
	Help: "The total number of messages sent",
})

func init() {
	prometheus.MustRegister(totalMessages)
	prometheus.MustRegister(totalReceived)
	prometheus.MustRegister(totalSent)
}

func main() {
	http.HandleFunc("/", relay)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}


func relay(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set up websocket connection:", err)
		return
	}
	defer conn.Close()

	// Register client
	var channelName string
	var channel *Channel
	var isSender, isReceiver bool

	// Wait for channel selection message
	messageType, messageBytes, err := conn.ReadMessage()
	if err != nil {
		fmt.Println("Failed to read message:", err)
		return
	}

	if messageType != websocket.TextMessage {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Invalid message type"))
		return
	}

	var data map[string]interface{}
	err = json.Unmarshal(messageBytes, &data)
	if err != nil {
		fmt.Println("Failed to parse message:", err)
		return
	}

	if data["command"] == "join" {
		channelName = data["channel"].(string)
		channelsMutex.Lock()
		if _, exists := channels[channelName]; !exists {
			channels[channelName] = &Channel{
				senders:   make(map[*websocket.Conn]struct{}),
				receivers: make(map[*websocket.Conn]struct{}),
			}
		}
		channel = channels[channelName]
		channelsMutex.Unlock()
		fmt.Printf("Client joined channel %s\n", channelName)
		conn.WriteJSON(map[string]interface{}{
			"type":    "STATUS",
			"message": "Joined channel",
		})
	} else {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Must join a channel first."))
		return
	}

	// Wait for role selection message
	for !isSender && !isReceiver {
		messageType, messageBytes, err = conn.ReadMessage()
		if err != nil {
			fmt.Println("Failed to read message:", err)
			return
		}

		if messageType != websocket.TextMessage {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Invalid message type"))
			return
		}

		err = json.Unmarshal(messageBytes, &data)
		if err != nil {
			fmt.Println("Failed to parse message:", err)
			return
		}

		if data["command"] == "select-role" {
			role := data["role"].(string)
			if role == "sender" {
				if len(channel.senders) == 0 {
					channel.senders[conn] = struct{}{}
					isSender = true
					fmt.Println("Client joined as sender")
				} else {
					conn.WriteJSON(map[string]interface{}{
						"type":    "ERROR",
						"message": "Cannot join as sender: sender already connected",
					})
				}
			} else if role == "receiver" {
				if len(channel.senders) == 0 {
					conn.WriteJSON(map[string]interface{}{
						"type":    "ERROR",
						"message": "Cannot join as receiver: no sender connected",
					})
				} else if len(channel.receivers) == 0 {
					channel.receivers[conn] = struct{}{}
					isReceiver = true

					conn.WriteJSON(map[string]interface{}{
						"status":    "ready",
					})
					for sender := range channel.senders {
						err = sender.WriteJSON(map[string]interface{}{
							"status":    "ready",
						})
						if err == nil {
							break
						}
						fmt.Println("No senders connected to forward message.")
					}
					fmt.Println("Client joined as receiver")
				} else {
					conn.WriteJSON(map[string]interface{}{
						"type":    "ERROR",
						"message": "Cannot join as receiver: receiver already connected",
					})
				}
			} else {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Invalid role selection."))
				return
			}
		}
	}

	// Receive message from sender or send message to receiver
	for {
		messageType, messageBytes, err = conn.ReadMessage()
		totalMessages.Inc()
		if err != nil {
			break
		}

		if messageType != websocket.TextMessage {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Invalid message type"))
			break
		}

		if isReceiver {
			// fmt.Println("Received message from sender:", string(messageBytes))

			// Forward message to sender
			// for sender := range channel.senders {
			// 	err = sender.WriteMessage(websocket.TextMessage, messageBytes)
			// 	if err == nil {
			// 		break
			// 	}
			// 	fmt.Println("No senders connected to forward message.")
			// }
		} else if isSender {
			if len(channel.receivers) == 0 {
				fmt.Println("No receivers connected, disconnecting sender.")
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "No receivers connected"))
				break
			}

			// Forward message to receiver
			for receiver := range channel.receivers {
				err = receiver.WriteMessage(websocket.TextMessage, messageBytes)
				if err == nil {
					totalSent.Inc()
					break
				}
				fmt.Println("No receivers connected to forward message.")
			}
		}
	}

	// Unregister client
	if isSender {
		delete(channel.senders, conn)
	} else if isReceiver {
		delete(channel.receivers, conn)
	}

	// Remove channel if empty
	if len(channel.senders) == 0 && len(channel.receivers) == 0 {
		channelsMutex.Lock()
		delete(channels, channelName)
		channelsMutex.Unlock()
		fmt.Printf("Channel %s removed.\n", channelName)
	}
}