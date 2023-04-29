package main

import (
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"encoding/base64"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/gorilla/websocket"
)

var (
	outputFile, _ = os.Create("received.bin")
)

type Message struct {
	Command string `json:"command"`
	Channel string `json:"channel,omitempty"`
	Role    string `json:"role,omitempty"`
}

type Metadata struct {
	ChunkNumber  int `json:"chunk_number"`
	ChunkSize    int `json:"chunk_size"`
	TotalChunks  int `json:"total_chunks"`
}

type RelayResponse struct {
	Data     string   `json:"data"`
	Metadata Metadata `json:"metadata"`
}


func colorize(str string, fg colorful.Color, bg colorful.Color) string {
	fgR, fgG, fgB := fg.RGB255()
	bgR, bgG, bgB := bg.RGB255()
	return fmt.Sprintf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm%s\033[0m", fgR, fgG, fgB, bgR, bgG, bgB, str)
}



func main() {
	serverAddress := url.URL{Scheme: "ws", Host: "99.33.36.109:8080"}

	conn, _, err := websocket.DefaultDialer.Dial(serverAddress.String(), nil)
	if err != nil {
		log.Fatalf("failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()

	messages := []Message{
		{
			Command: "join",
			Channel: "example",
		},
		{
			Command: "select-role",
			Role:    "receiver",
		},
	}

	err = conn.WriteJSON(messages[0])
		if err != nil {
			log.Printf("failed to send message: %v", err)
			os.Exit(1)
		}

		_, response, err := conn.ReadMessage()
		if err != nil {
			log.Printf("failed to read message from server: %v", err)
			os.Exit(1)
		}
		fmt.Println(string(response))

		err = conn.WriteJSON(messages[1])
		if err != nil {
			log.Printf("failed to send message: %v", err)
			os.Exit(1)
		}
		

		_, response, err = conn.ReadMessage()
		if err != nil {
			log.Printf("failed to read message from server: %v", err)
			os.Exit(1)
		}
		fmt.Println(string(response))
		if strings.Contains(string(response), "ERROR") {
			fmt.Println("No senders yet!")
			conn.Close()
			os.Exit(0)
		}
	var receivedChunks int
	lastPercent := 0
	for {
		_, response, err := conn.ReadMessage()
		if err != nil {
			log.Printf("failed to read message from server: %v", err)
			os.Exit(1)
		}

		var RelayResponse RelayResponse
		err = json.Unmarshal(response, &RelayResponse)

		decodedData, err := base64.StdEncoding.DecodeString(RelayResponse.Data)
		if err != nil {
			log.Printf("failed to decode Base64 data: %v", err)
			os.Exit(1)
		}

		_, err = outputFile.Write(decodedData)
		if err != nil {
			log.Printf("failed to write data to file: %v", err)
			os.Exit(1)
		}



		from, _ := colorful.Hex("#FFA500")
		to, _ := colorful.Hex("#FF4500")

		// Define the progress bar width
		width := 50

		// Define the total number of iterations
		total := RelayResponse.Metadata.TotalChunks

	// Loop through the iterations
		for i := 0; i <= total; i++ {
			// Calculate the percentage
			percent := (receivedChunks * 100 / total)
			if lastPercent == percent {
				break
			} else {
				lastPercent = percent
			}

			// Create the progress bar string
			barWidth := width * percent / 100
			bar := ""

			for j := 0; j < barWidth; j++ {
				// Calculate the color at this position in the gradient
				fraction := float64(j) / float64(width-1)
				color := from.BlendHcl(to, fraction).Clamped()

				// Calculate the inverted percentage for the background color
				invertedFraction := 1 - float64(j)/float64(width-1)
				bgColor := from.BlendHcl(to, invertedFraction).Clamped()

				// Add the progress bar character with the color
				bar += colorize("█", color, bgColor)
			}

			bar += colorize("▋", from.BlendHcl(to, float64(barWidth) / float64(width-1)).Clamped(), from.BlendHcl(to, float64(barWidth) / float64(width-1)).Clamped())

			bar += strings.Repeat(" ", width-barWidth)

			bar += " "
			bar += fmt.Sprintf("[ %d%% ]", percent)

			// Print the progress bar
			fmt.Printf("\r%s", bar)
		}


		if receivedChunks == RelayResponse.Metadata.TotalChunks {
			fmt.Println(" All chunks received successfully!")
			break
		}
		receivedChunks++
	}
}
