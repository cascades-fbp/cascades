package log

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/daviddengcn/go-colortext"
	"sync"
)

type LogFactory struct {
	Logs    map[string]*Log
	Padding int
	Name    string
}

type Log struct {
	Name    string
	Color   ct.Color
	IsError bool
	Factory *LogFactory
}

var mx, sysMx sync.Mutex

var colors = []ct.Color{
	ct.Cyan,
	ct.Yellow,
	ct.Green,
	ct.Magenta,
	ct.Red,
	ct.Blue,
}

// Default factory for creating loggers
var DefaultFactory *LogFactory

// Log factory constructor
func NewLogFactory() (of *LogFactory) {
	of = new(LogFactory)
	of.Logs = make(map[string]*Log)
	return
}

func (o *Log) Println(str string) {
	o.Write([]byte(str))
}

// Safe (using mutex) writing to a specific log
func (o *Log) Write(b []byte) (num int, err error) {
	mx.Lock()
	defer mx.Unlock()
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		formatter := fmt.Sprintf("%%-%ds | ", o.Factory.Padding)
		ct.ChangeColor(o.Color, true, ct.None, false)
		fmt.Printf(formatter, o.Name)
		if o.IsError {
			ct.ChangeColor(ct.Red, true, ct.None, true)
		} else {
			ct.ResetColor()
		}
		fmt.Println(scanner.Text())
		ct.ResetColor()
	}
	num = len(b)
	return
}

func (of *LogFactory) CreateLog(name string, index int, isError bool) *Log {
	of.Logs[name] = &Log{name, colors[index%len(colors)], isError, of}
	return of.Logs[name]
}

// Safe (using mutex) write to output (from system's name)
func (of *LogFactory) SystemOutput(str string) {
	sysMx.Lock()
	defer sysMx.Unlock()
	ct.ChangeColor(ct.White, true, ct.None, false)
	formatter := fmt.Sprintf("%%-%ds | ", of.Padding)
	fmt.Printf(formatter, of.Name)
	ct.ResetColor()
	fmt.Println(str)
	ct.ResetColor()
}

// Safe (using mutex) write to error output (from system's name)
func (of *LogFactory) ErrorOutput(str string) {
	sysMx.Lock()
	defer sysMx.Unlock()
	fmt.Printf("ERROR: %s\n", str)
}

func init() {
	DefaultFactory = NewLogFactory()
	DefaultFactory.Name = "runtime"
	DefaultFactory.Padding = len(DefaultFactory.Name)
}

func SystemOutput(str string) {
	DefaultFactory.SystemOutput(str)
}

func ErrorOutput(str string) {
	DefaultFactory.ErrorOutput(str)
}
