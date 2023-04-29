package main

import (
	"fmt"
	"log"
    "encoding/base64"
	"encoding/json"
	"os"
    "net/url"
    "strings"

	"github.com/gorilla/websocket"
    "github.com/lucasb-eyer/go-colorful"
)

func waitForStatus(conn *websocket.Conn, expectedStatus string) error {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var response map[string]interface{}
		err = json.Unmarshal(message, &response)
		if err != nil {
			return err
		}

		status, ok := response["status"].(string)
		if ok && status == expectedStatus {
			break
		}
	}
	return nil
}

func colorize(str string, fg colorful.Color, bg colorful.Color) string {
	fgR, fgG, fgB := fg.RGB255()
	bgR, bgG, bgB := bg.RGB255()
	return fmt.Sprintf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm%s\033[0m", fgR, fgG, fgB, bgR, bgG, bgB, str)
}

func main() {
    u := url.URL{Scheme: "ws", Host: "99.33.36.109:8080", Path: "/"}
    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer conn.Close()

	// Join the channel and select the sender role
	initialMessages := []string{
		`{"command": "join", "channel": "example"}`,
		`{"command": "select-role", "role": "sender"}`,
	}

	for _, message := range initialMessages {
		err = conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("Failed to send message:", err)
			return
		}
	}


    fileName := "data.bin"
	chunkSize := 1024 * 1024 * 5 // in bytes

	// Open the binary file for reading
	binaryFile, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer binaryFile.Close()

	// Get the total size of the binary file
	fileInfo, err := binaryFile.Stat()
	if err != nil {
		panic(err)
	}
	totalSize := fileInfo.Size()

	// Calculate the total number of chunks
	numChunks := ((int(totalSize) + int(chunkSize) - 1) / int(chunkSize)) // Round up to the nearest integer

    err = waitForStatus(conn, "ready")
	if err != nil {
		log.Println("Failed to receive expected status:", err)
		return
	}

     lastPercent := 0

	for chunkNumber := int(0); chunkNumber < numChunks; chunkNumber++ {
		// Read the next chunk of data
		chunkData := make([]byte, chunkSize)
		n, err := binaryFile.Read(chunkData)
		if err != nil {
			panic(err)
		}
		chunkData = chunkData[:n]

		// Construct the JSON metadata object
		metadata := map[string]interface{}{
			"chunk_number": chunkNumber,
			"chunk_size":   len(chunkData),
			"total_chunks": numChunks - 1,
		}

		// Construct the JSON message object with the metadata and binary data
		data := base64.StdEncoding.EncodeToString(chunkData)
		message := map[string]interface{}{
			"metadata": metadata,
			"data":     data,
		}

		// Encode the message as JSON
		messageJSON, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}

        from, _ := colorful.Hex("#FFA500")
	to, _ := colorful.Hex("#FF4500")

	// Define the progress bar width
	width := 50

	// Define the total number of iterations
	total := numChunks

	// Loop through the iterations
	for i := 0; i <= total; i++ {
		// Calculate the percentage
		percent := int(chunkNumber * 100 / total) + 1
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

		// Add the message to the list of chunks
		// chunks = append(chunks, string(messageJSON))

		// Print the metadata for debugging purposes
		// fmt.Println("Sent message: ", metadata)
        err = conn.WriteMessage(websocket.TextMessage, []byte(messageJSON))
		if err != nil {
			log.Println("Failed to send message:", err)
			return
		}
	}
    fmt.Println(" All chunks sent succesfully!")
}
