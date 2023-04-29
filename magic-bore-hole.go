package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "magic-borehole",
		Usage: "A self hosted method of magically teleporting files and folders from one computer to another",
		Commands: []*cli.Command{
			{
				Name:    "send",
				Aliases: []string{"s"},
				Usage:   "Send one or more files or folders from another computer",
				Action:  func(c *cli.Context) error {
					// Execute the foo function...
					fmt.Println("Executing the send function...")
					return nil
				},
			},
			{
				Name:    "receive",
				Aliases: []string{"r"},
				Usage:   "Receive one or more files or folders from another computer",
				Action:  func(c *cli.Context) error {
					// Execute the bar function...
					fmt.Println("Executing the receive function...")
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
