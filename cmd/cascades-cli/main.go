package main

import (
	"github.com/spf13/cobra"
)

var (
	indexFilepath string
	forceCommand  bool
	tcpPort       uint
	content       string
	endpoint      string
	debug         bool
	dryRun        bool
)

func main() {
	var cmdRoot = &cobra.Command{
		Use:   "cascades-cli",
		Short: "cascades-cli is a CLI for executing .fbp/.json graphs or submitting them for execution to remote runtime.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}

	cmdRoot.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug extra output")

	cmdRegister.PersistentFlags().StringVarP(&indexFilepath, "index", "i", "conf/registry.json", "File path to JSON components index")
	cmdRegister.PersistentFlags().BoolVarP(&forceCommand, "force", "f", false, "Force command execution")
	cmdRegisterFile.PersistentFlags().StringP("name", "n", "", "Name of the component in the registry")
	cmdRegister.AddCommand(cmdRegisterFile, cmdRegisterDir)
	cmdRoot.AddCommand(cmdRegister)

	cmdRun.PersistentFlags().StringVarP(&indexFilepath, "index", "i", "conf/registry.json", "File path to JSON components index")
	cmdRun.PersistentFlags().UintVarP(&tcpPort, "port", "p", 5555, "Initial port to use for connections")
	cmdRun.PersistentFlags().BoolVarP(&dryRun, "dry", "r", false, "Dry run (parse nework but don't execute it)")
	cmdRoot.AddCommand(cmdRun)

	cmdIIP.PersistentFlags().StringVarP(&content, "content", "c", "", "Content to send as IIP to a given port")
	cmdIIP.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "", "Endpoint of the port to send IIP to")
	cmdRoot.AddCommand(cmdIIP)

	cmdRoot.Execute()
}
