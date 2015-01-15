package runtime

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cascades-fbp/cascades/graph"
	"github.com/cascades-fbp/cascades/library"
	"github.com/cascades-fbp/cascades/log"
	zmq "github.com/pebbe/zmq4"
)

var (
	procWaitGroup sync.WaitGroup
	shutdownMutex sync.Mutex
)

//
// Runtime structure corresponds to a single network
//
type Runtime struct {
	registrar      library.Registrar
	initialTCPPort uint
	graph          *graph.Description
	processes      map[string]*Process
	iips           []ProcessIIP
	Done           chan bool
	Debug          bool
}

//
// NewRuntime is a Runtime constructor
//
func NewRuntime(registrar library.Registrar, initialTCPPort uint) *Runtime {
	r := &Runtime{
		registrar:      registrar,
		initialTCPPort: initialTCPPort,
		processes:      map[string]*Process{},
		iips:           []ProcessIIP{},
		Done:           make(chan bool),
		Debug:          false,
	}
	return r
}

//
// LoadGraph loads graph definition in supported format from a given file path
//
func (r *Runtime) LoadGraph(graphfile string) error {
	var err error
	if r.graph, err = loadGraph(graphfile); err != nil {
		return err
	}
	err = r.flattenGraph(r.graph)
	return err
}

//
// Validates the graph against library and flattens it (unwraps subgraphs)
//
func (r *Runtime) flattenGraph(g *graph.Description) error {
	// copy processes map to interate over
	hasSubgraphs := false
	processes := g.Processes
	for name, process := range processes {
		// Check if known component
		e, err := r.registrar.Get(process.Component)
		if err != nil {
			return fmt.Errorf("Component %s not found in the library", process.Component)
		}

		// Check if subgraph
		if !strings.HasSuffix(e.Executable, ".fbp") && !strings.HasSuffix(e.Executable, ".json") {
			continue
		}

		// Load subgraph & "unwrap" it
		hasSubgraphs = true
		subgraph, err := loadGraph(e.Executable)
		if err != nil {
			return err
		}

		// Replace subgraph with its processes/connections in the graph
		delete(g.Processes, name)
		for n, p := range subgraph.Processes {
			g.Processes[name+n] = p
		}
		for _, c := range subgraph.Connections {
			if c.Src != nil {
				c.Src.Process = name + c.Src.Process
			}
			c.Tgt.Process = name + c.Tgt.Process
			g.Connections = append(g.Connections, c)
		}
		for _, e := range subgraph.Inports {
			connections := []graph.Connection{}
			parts := strings.SplitN(e.Private, ".", 2)
			for _, c := range g.Connections {
				if c.Tgt.Process == name && c.Tgt.Port == e.Public {
					c.Tgt.Process = name + parts[0]
					c.Tgt.Port = parts[1]
				}
				connections = append(connections, c)
			}
			g.Connections = connections
		}
		for _, e := range subgraph.Outports {
			connections := []graph.Connection{}
			parts := strings.SplitN(e.Private, ".", 2)
			for _, c := range g.Connections {
				if c.Src != nil && c.Src.Process == name && c.Src.Port == e.Public {
					c.Src.Process = name + parts[0]
					c.Src.Port = parts[1]
				}
				connections = append(connections, c)
			}
			g.Connections = connections
		}

	}

	if hasSubgraphs {
		err := r.flattenGraph(g)
		if err != nil {
			return err
		}
	}

	return nil
}

//
// PrintGraph print the current graph for debug purposes
//
func (r *Runtime) PrintGraph() {
	fmt.Println("--------- Properties ----------")
	for k, v := range r.graph.Properties {
		fmt.Printf("%s: %s\n", k, v)
	}
	fmt.Println("---------- Inports -----------")
	for _, e := range r.graph.Inports {
		fmt.Printf("%s exposed as %s", e.Private, e.Public)
	}
	fmt.Println("---------- Outports -----------")
	for _, e := range r.graph.Outports {
		fmt.Printf("%s exposed as %s", e.Private, e.Public)
	}
	fmt.Println("---------- Processes ----------")
	for p, c := range r.graph.Processes {
		fmt.Println(p, c.String())
	}
	fmt.Println("--------- Connections ---------")
	for _, c := range r.graph.Connections {
		fmt.Println(c.String())
	}
	fmt.Println("-------------------------------")
}

//
// Prepare processes for start using graph definition
//
func (r *Runtime) prepareProcesses() error {
	// Create process structures for execution
	nameLength := log.DefaultFactory.Padding
	for name, p := range r.graph.Processes {
		entry, err := r.registrar.Get(p.Component)
		if err != nil {
			return err
		}
		r.processes[name] = NewProcess(entry.Executable)
		if r.Debug {
			r.processes[name].Args["--debug"] = ""
		}
		if len(name) > nameLength {
			nameLength = len(name)
		}
	}
	// Create ZMQ sockets for each unique port
	var currentPort = r.initialTCPPort
	var (
		endpoint, srcEndpoint, tgtEndpoint string
		index, srcIndex, tgtIndex          int
	)
	sockets := map[string]string{}
	for _, c := range r.graph.Connections {
		if c.Src == nil {
			iip := ProcessIIP{
				Payload: c.Data,
			}
			index = 0
			if c.Tgt.Index != nil {
				index = *c.Tgt.Index
			}
			endpoint = fmt.Sprintf("%s.%s.%v", c.Tgt.Process, c.Tgt.Port, index)
			if s, ok := sockets[endpoint]; ok {
				iip.Socket = s
			} else {
				s = fmt.Sprintf("tcp://127.0.0.1:%v", currentPort)
				currentPort++
				iip.Socket = s
				sockets[endpoint] = s
			}
			r.iips = append(r.iips, iip)
		} else {
			srcIndex = 0
			tgtIndex = 0
			if c.Src.Index != nil {
				srcIndex = *c.Src.Index
			}
			if c.Tgt.Index != nil {
				tgtIndex = *c.Tgt.Index
			}
			srcEndpoint = fmt.Sprintf("%s.%s.%v", c.Src.Process, c.Src.Port, srcIndex)
			tgtEndpoint = fmt.Sprintf("%s.%s.%v", c.Tgt.Process, c.Tgt.Port, tgtIndex)

			if s, ok := sockets[srcEndpoint]; ok {
				if _, ok := sockets[tgtEndpoint]; !ok {
					sockets[tgtEndpoint] = s
				}
			} else {
				if s, ok := sockets[tgtEndpoint]; ok {
					sockets[srcEndpoint] = s
				} else {
					s = fmt.Sprintf("tcp://127.0.0.1:%v", currentPort)
					currentPort++
					sockets[srcEndpoint] = s
					sockets[tgtEndpoint] = s
				}
			}
		}
	}
	// Compact sockets
	arguments := map[string][]string{}
	for n, s := range sockets {
		parts := strings.SplitN(n, ".", 3)
		k := parts[0] + "." + parts[1]
		if _, ok := arguments[k]; ok {
			arguments[k] = append(arguments[k], s)
		} else {
			arguments[k] = []string{s}
		}

	}
	// Add sockets to component CLI arguments
	for n, s := range arguments {
		parts := strings.SplitN(n, ".", 2)
		r.processes[parts[0]].Args["--port."+strings.ToLower(parts[1])] = strings.Join(s, ",")
		if r.Debug {
			fmt.Println(n, s)
		}
	}

	if r.Debug {
		fmt.Println("--------- Executables ---------")
		for n, p := range r.processes {
			fmt.Printf("%s: %#v %#v\n", n, p.Executable, p.Arguments())
		}
		fmt.Println("-------------------------------")
	}

	log.DefaultFactory.Padding = nameLength

	return nil
}

//
// Start the network based on the current graph
//
func (r *Runtime) Start() {
	err := r.prepareProcesses()
	if err != nil {
		log.ErrorOutput("Failed to create a process: " + err.Error())
		r.Done <- true
		return
	}

	if len(r.processes) == 0 {
		log.SystemOutput("No processes to start")
		r.Done <- true
		return
	}

	log.SystemOutput("Starting processes...")
	idx := 0
	for name, ps := range r.processes {
		shutdownMutex.Lock()
		procWaitGroup.Add(1)

		ps.Stdin = nil
		ps.Stdout = log.DefaultFactory.CreateLog(name, idx, false)
		ps.Stderr = log.DefaultFactory.CreateLog(name, idx, true)
		ps.Start()

		go func(name string, ps *Process) {
			ps.Wait()
			procWaitGroup.Done()
			delete(r.processes, name)
			fmt.Fprintln(ps.Stdout, "Stopped")

			// Shutdown when no processes left, otherwise network should collapse
			// in a cascade way...
			//if !ps.cmd.ProcessState.Success() || len(r.processes) == 0 {
			if len(r.processes) == 0 {
				fmt.Fprintln(ps.Stdout, "I was the last running process. Calling runtime to SHUTDOWN")
				r.Shutdown()
			}

		}(name, ps)

		shutdownMutex.Unlock()

		idx++
	}

	r.Activate()

	procWaitGroup.Wait()
}

//
// Activate network by sending out all IIPs
//
func (r *Runtime) Activate() {
	if len(r.iips) > 0 {
		log.SystemOutput("Activating processes by sending IIPs...")

		// Connect to ports of IIP (so the components can resume execution)
		senders := make([]*zmq.Socket, len(r.iips))
		for i, iip := range r.iips {
			senders[i], _ = zmq.NewSocket(zmq.PUSH)
			senders[i].Connect(iip.Socket)
		}
		defer func() {
			for i := range senders {
				senders[i].Close()
			}
		}()

		// Give some time to the network to deploy
		//TODO: replace this with info from processes (check all entries in processes hash map)
		time.Sleep(2 * time.Second)

		// Send IIPs out!
		for i, iip := range r.iips {
			log.SystemOutput(fmt.Sprintf("Sending '%s' to socket '%s'", iip.Payload, iip.Socket))
			senders[i].SendMessageDontwait(NewPacket([]byte(iip.Payload)))
		}
	}
}
