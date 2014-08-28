package runtime

import (
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/cascades-fbp/cascades/graph"
	"github.com/cascades-fbp/cascades/library"
	"github.com/cascades-fbp/cascades/log"
	"strings"
	"sync"
	"time"
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
	graph          *graph.GraphDescription
	processes      map[string]*Process
	iips           []ProcessIIP
	Done           chan bool
	Debug          bool
}

//
// Runtime constructor
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
// Loads graph definition in supported format from a given file path
//
func (self *Runtime) LoadGraph(graphfile string) error {
	var err error
	if self.graph, err = loadGraph(graphfile); err != nil {
		return err
	}
	err = self.flattenGraph(self.graph)
	return err
}

//
// Validates the graph against library and flattens it (unwraps subgraphs)
//
func (self *Runtime) flattenGraph(g *graph.GraphDescription) error {
	// copy processes map to interate over
	hasSubgraphs := false
	processes := g.Processes
	for name, process := range processes {
		// Check if known component
		e, err := self.registrar.Get(process.Component)
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
		err := self.flattenGraph(g)
		if err != nil {
			return err
		}
	}

	return nil
}

//
// Print the current graph for debug purposes
//
func (self *Runtime) PrintGraph() {
	fmt.Println("--------- Properties ----------")
	for k, v := range self.graph.Properties {
		fmt.Printf("%s: %s\n", k, v)
	}
	fmt.Println("---------- Inports -----------")
	for _, e := range self.graph.Inports {
		fmt.Printf("%s exposed as %s", e.Private, e.Public)
	}
	fmt.Println("---------- Outports -----------")
	for _, e := range self.graph.Outports {
		fmt.Printf("%s exposed as %s", e.Private, e.Public)
	}
	fmt.Println("---------- Processes ----------")
	for p, c := range self.graph.Processes {
		fmt.Println(p, c.String())
	}
	fmt.Println("--------- Connections ---------")
	for _, c := range self.graph.Connections {
		fmt.Println(c.String())
	}
	fmt.Println("-------------------------------")
}

//
// Prepare processes for start using graph definition
//
func (self *Runtime) prepareProcesses() error {
	// Create process structures for execution
	nameLength := log.DefaultFactory.Padding
	for name, p := range self.graph.Processes {
		entry, err := self.registrar.Get(p.Component)
		if err != nil {
			return err
		}
		self.processes[name] = NewProcess(entry.Executable)
		if self.Debug {
			self.processes[name].Args["-debug"] = "true"
		}
		if len(name) > nameLength {
			nameLength = len(name)
		}
	}
	// Create ZMQ sockets for each unique port
	var currentPort = self.initialTCPPort
	var (
		endpoint, srcEndpoint, tgtEndpoint string
		index, srcIndex, tgtIndex          int
	)
	sockets := map[string]string{}
	for _, c := range self.graph.Connections {
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
			self.iips = append(self.iips, iip)
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
		self.processes[parts[0]].Args["-port."+strings.ToLower(parts[1])] = strings.Join(s, ",")
		if self.Debug {
			fmt.Println(n, s)
		}
	}

	if self.Debug {
		fmt.Println("--------- Executables ---------")
		for n, p := range self.processes {
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
func (self *Runtime) Start() {
	err := self.prepareProcesses()
	if err != nil {
		log.ErrorOutput("Failed to create a process: " + err.Error())
		self.Done <- true
		return
	}

	if len(self.processes) == 0 {
		log.SystemOutput("No processes to start")
		self.Done <- true
		return
	}

	log.SystemOutput("Starting processes...")
	idx := 0
	for name, ps := range self.processes {
		shutdownMutex.Lock()
		procWaitGroup.Add(1)

		ps.Stdin = nil
		ps.Stdout = log.DefaultFactory.CreateLog(name, idx, false)
		ps.Stderr = log.DefaultFactory.CreateLog(name, idx, true)
		ps.Start()

		go func(name string, ps *Process) {
			ps.Wait()
			procWaitGroup.Done()
			delete(self.processes, name)
			fmt.Fprintln(ps.Stdout, "Stopped")
			//TODO: check all connections with this process and send them shutdown commands as well
			if !ps.cmd.ProcessState.Success() {
				self.Shutdown()
			}

		}(name, ps)

		shutdownMutex.Unlock()

		idx++
	}

	self.Activate()

	procWaitGroup.Wait()
}

//
// Activate network by sending out all IIPs
//
func (self *Runtime) Activate() {
	if len(self.iips) > 0 {
		// Give some time to the network to deploy
		//TODO: replace this with info from processes (check all entries in processes hash map)
		time.Sleep(1 * time.Second)

		log.SystemOutput("Activating processes by sending IIPs...")

		context, _ := zmq.NewContext()
		defer context.Close()

		sender, _ := context.NewSocket(zmq.PUSH)
		defer sender.Close()

		for _, iip := range self.iips {
			log.SystemOutput(fmt.Sprintf("Sending '%s' to socket '%s'", iip.Payload, iip.Socket))
			sender.Connect(iip.Socket)
			sender.SendMultipart(NewPacket([]byte(iip.Payload)), zmq.NOBLOCK)
			sender.Disconnect(iip.Socket)
		}
	}
}
