package runtime

/*
func ShutdownProcesses(of *OutletFactory) {
	shutdown_mutex.Lock()
	of.SystemOutput("shutting down")
	for name, ps := range processes {
		of.SystemOutput(fmt.Sprintf("terminating %s", name))
		ps.cmd.Process.Signal(os.Kill)
	}
	os.Exit(1)
}
*/
