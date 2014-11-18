package fbp

import (
	"testing"
)

const (
	graphIIP string = `
	'5s'
	`
	graphTickLogger string = `
	'5s' -> INTERVAL Ticker(core/ticker) OUT -> IN Forward(core/passthru)
	Forward OUT -> IN Log(core/console)
	`
	graphOneLiner string = `
	Demo OUT -> IN Process RESULT -> INPUT Visualize DISPLAY -> IN Console LOG -> IN D1
	Console ERR -> IN D2
	`
	graphDemo string = `
	'somefile.txt' -> SOURCE Read(ReadFile:main)
	Read() OUT -> IN Split(SplitStr:main)
	Split() OUT -> IN Count(Counter:main)
	Count() COUNT -> IN Display(Output:main)
	Read() ERROR -> IN Display()
	`
	graphExportedInPort = `
	INPORT=Read.IN:FILENAME
	INPORT=Read.OPTIONS:CONFIG
	OUTPORT=Process.OUT:RESULT
	Read(ReadFile) OUT -> IN Process(Output)
	`

	graphArrayPorts string = `
	'pattern1' -> IN[0] Router(router)
	Router OUT[0] -> IN Log1(console)
	'pattern2' -> IN[1] Router
	Router OUT[1] -> IN Log2(console)
	'pattern3' -> IN[2] Router
	Router OUT[2] -> IN Log3(console)
	'some data' -> DATA Router
 	`

	graphExportedArrayPort = `
	INPORT=Read.IN:FILENAME
	INPORT=Read.OPTIONS:CONFIG
	INPORT=Process.IN[0]:EXTRA
	OUTPORT=Process.OUT[1]:RESULT
	Read(ReadFile) OUT -> IN[0] Process(Output)
	Process OUT[0] -> IN Log(Console)
	`

	graphArrayPortsOneline string = `
	'pattern1' -> IN[0] Router(router) OUT[0] -> IN Log(console)
 	`
)

func testGraph(t *testing.T, graph string) *Fbp {
	parser := &Fbp{Buffer: graph}
	parser.Init()
	err := parser.Parse()
	if err != nil {
		t.Log(err.Error())
		t.Fail()
	}
	parser.Execute()
	if err = parser.Validate(); err != nil {
		t.Log(err.Error())
		t.Fail()
	}

	t.Log("------------ Processes --------------")
	for _, p := range parser.Processes {
		t.Logf("%#v", p.String())
	}
	t.Log("----------- Connections -------------")
	for _, c := range parser.Connections {
		t.Logf("%#v", c.String())
	}
	t.Log("----------- Inports -------------")
	for k, p := range parser.Inports {
		t.Logf("%s: %#v", k, p.String())
	}
	t.Log("----------- Outports -------------")
	for k, p := range parser.Outports {
		t.Logf("%s: %#v", k, p.String())
	}
	return parser
}

func TestGraphIIP(t *testing.T) {
	testGraph(t, graphIIP)
}

func TestGraphTickLogger(t *testing.T) {
	parser := testGraph(t, graphTickLogger)
	if len(parser.Processes) != 3 {
		t.Fatal("Should be only 3 processes")
	}
	if len(parser.Connections) != 3 {
		t.Fatal("Should be only 3 connections")
	}
}

func TestGraphGraphOneLiner(t *testing.T) {
	parser := testGraph(t, graphOneLiner)
	if len(parser.Processes) != 0 {
		t.Fatal("Should be only 0 processes")
	}
	if len(parser.Connections) != 5 {
		t.Fatal("Should be only 5 connections")
	}
}

func TestGraphDemo(t *testing.T) {
	parser := testGraph(t, graphDemo)
	if len(parser.Processes) != 4 {
		t.Fatal("Should be only 4 processes")
	}
	if len(parser.Connections) != 5 {
		t.Fatal("Should be only 5 connections")
	}
}

func TestGraphExportedInPort(t *testing.T) {
	parser := testGraph(t, graphExportedInPort)
	if len(parser.Processes) != 2 {
		t.Fatal("Should be only 2 processes")
	}
	if len(parser.Connections) != 1 {
		t.Fatal("Should be only 1 connections")
	}
	if len(parser.Inports) != 2 {
		t.Fatal("Should be only 2 inports")
	}
	if len(parser.Outports) != 1 {
		t.Fatal("Should be only 1 outports")
	}
}

func TestGraphArrayPorts(t *testing.T) {
	parser := testGraph(t, graphArrayPorts)
	if len(parser.Processes) != 4 {
		t.Fatal("Should be only 4 processes")
	}
	if len(parser.Connections) != 7 {
		t.Fatal("Should be only 7 connections")
	}
}

func TestGraphExportedArrayPort(t *testing.T) {
	testGraph(t, graphExportedArrayPort)
}

func TestGraphArrayPortsOneline(t *testing.T) {
	parser := testGraph(t, graphArrayPortsOneline)
	if len(parser.Processes) != 4 {
		t.Fatal("Should be only 4 processes")
	}
	if len(parser.Connections) != 7 {
		t.Fatal("Should be only 7 connections")
	}
}
