package main

import (
	"log"
	"os"
	"time"

	"github.com/knadh/pfxsigner/internal/processor"
	"github.com/urfave/cli"
)

var (
	buildString = ""
	proc        *processor.Processor
	logger      *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
}

func main() {
	app := cli.NewApp()
	app.Name = "pfxsigner"
	app.Usage = "utility for signing PDFs with PFX signatures"
	app.Version = buildString
	app.Action = func(c *cli.Context) error {
		log.Println("no action to run. bye.")
		return nil
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "pfx-file", Value: "cert.pfx", Usage: "path to PFX file", TakesFile: true},
		cli.StringFlag{Name: "pfx-password", Value: "", Usage: "PFX password"},
		cli.StringFlag{Name: "password", Value: "", Usage: "nest password"},
		cli.StringFlag{Name: "props-file", Value: "props.json", Usage: "path to the JSON file with default signature properties", TakesFile: true},
	}
	app.Commands = []cli.Command{
		// Request-response mode.
		cli.Command{
			Name:        "cli",
			Description: "run the utility in CLI mode",
			Flags: []cli.Flag{
				cli.IntFlag{Name: "workers", Value: 2,
					Usage: "number of workers to run for signing"},
			},
			Action: initApp(initCLI),
		},

		// Stream mode.
		cli.Command{
			Name:        "server",
			Description: "run the utility in HTTP server mode",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "address", Value: ":8000",
					Usage: "address to listen on"},
				cli.DurationFlag{Name: "timeout", Value: time.Second * 30,
					Usage: "request timeout (eg: 10s)"},
			},
			Action: initApp(initServer),
		},
	}
	app.Run(os.Args)
}
