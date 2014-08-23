package main

import (
	"github.com/codegangsta/cli"
)

func serve(c *cli.Context) {
	println("serve:", c.Args().First())
}
