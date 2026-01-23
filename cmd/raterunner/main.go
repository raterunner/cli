package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "raterunner",
		Usage:   "Raterunner CLI application",
		Version: "0.1.0",
		Commands: []*cli.Command{
			{
				Name:    "hello",
				Aliases: []string{"hi"},
				Usage:   "Say hello",
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if name == "" {
						name = "World"
					}
					fmt.Printf("Hello, %s!\n", name)
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
