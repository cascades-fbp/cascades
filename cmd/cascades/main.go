package main

import (
	"os"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "cascades"
	app.Usage = "A Cascades FBP runtime/scheduler for the FBP applications."
	app.Author = "Alexander Lobunets"
	app.Email = "alexander.lobunets@gmail.com"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file, f",
			Value: "library.json",
			Usage: "components library file",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "enable extra output for debug purposes",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Runs a given graph defined in the .fbp or .json formats",
			Action: run,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "port, p",
					Value: 5000,
					Usage: "initial port to use for connections between nodes",
				},
				cli.BoolFlag{
					Name:  "dry",
					Usage: "dry run (parses and validates the graph, exits without executing it)",
				},
			},
		},
		{
			Name:  "library",
			Usage: "Manage a library of components",
			Subcommands: []cli.Command{
				{
					Name:   "add",
					Usage:  "updates a library with component(s) from a given path (either directory with components  or component file)",
					Action: addToLibrary,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "name",
							Value: "",
							Usage: "component's name if adding a component file (ignored if adding a directory",
						},
						cli.BoolFlag{
							Name:  "force",
							Usage: "enforces updating a component entry in the library if it already exists",
						},
					},
				},
				{
					Name:   "list",
					Usage:  "lists all registered components and their documentation",
					Action: listLibrary,
				},
				{
					Name:   "info",
					Usage:  "prints details for a given component",
					Action: infoFromLibrary,
				},
			},
		},
		/*
			{
				Name:  "serve",
				Usage: "Start a runtime server for executing submitted graphs",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "addr",
						Value: "0.0.0.0:7878",
						Usage: "binding address for the server",
					},
					cli.StringFlag{
						Name:  "static",
						Value: "static",
						Usage: "root directory with static resources (will be mounted as /static/)",
					},
				},
				Action: serve,
			},
		*/
	}

	app.Run(os.Args)
}
