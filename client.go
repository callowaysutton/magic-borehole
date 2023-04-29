package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "magic-bore-hole",
		Usage: "A self hosted way to teleport files magically from one place to another",
		Action: func(c *cli.Context) error {
			fmt.Println("Hello, World!")
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "config.yaml",
				Usage: "path to config file",
			},
		},
		ArgsUsage: "<file>",
		UsageText: "myapp send [--flags] <file(s) | directory>\nmyapp receive [--flags] <code>",
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
