package log

import (
	"bufio"
	"bytes"
	"fmt"
	"sync"

	"github.com/daviddengcn/go-colortext"
)

// Factory is a factory of individual logs
type Factory struct {
	Logs    map[string]*Log
	Padding int
	Name    string
}

// Log represents a named colorful logger
type Log struct {
	Name    string
	Color   ct.Color
	IsError bool
	Factory *Factory
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
var DefaultFactory *Factory

// NewFactory is a Log factory constructor
func NewFactory() (of *Factory) {
	of = new(Factory)
	of.Logs = make(map[string]*Log)
	return
}

// Println writes a given string to logger's stream
func (o *Log) Println(str string) {
	o.Write([]byte(str))
}

// Write safely (using mutex) to a specific log
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

// CreateLog create a new Log structure
func (of *Factory) CreateLog(name string, index int, isError bool) *Log {
	of.Logs[name] = &Log{name, colors[index%len(colors)], isError, of}
	return of.Logs[name]
}

// SystemOutput prints a given string safely (using mutex) to output (from system's name)
func (of *Factory) SystemOutput(str string) {
	sysMx.Lock()
	defer sysMx.Unlock()
	ct.ChangeColor(ct.White, true, ct.None, false)
	formatter := fmt.Sprintf("%%-%ds | ", of.Padding)
	fmt.Printf(formatter, of.Name)
	ct.ResetColor()
	fmt.Println(str)
	ct.ResetColor()
}

// ErrorOutput writes safely (using mutex) to error output (from system's name)
func (of *Factory) ErrorOutput(str string) {
	sysMx.Lock()
	defer sysMx.Unlock()
	fmt.Printf("ERROR: %s\n", str)
}

func init() {
	DefaultFactory = NewFactory()
	DefaultFactory.Name = "runtime"
	DefaultFactory.Padding = len(DefaultFactory.Name)
}

// SystemOutput writes output using default factory
func SystemOutput(str string) {
	DefaultFactory.SystemOutput(str)
}

// ErrorOutput writes error using default factory
func ErrorOutput(str string) {
	DefaultFactory.ErrorOutput(str)
}
