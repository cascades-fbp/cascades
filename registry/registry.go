package registry

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	NotFound = errors.New("Component not found")
)

//
// Defines the interface of components registry
//
type Registrar interface {
	Add(entry Entry)
	Exists(name string) bool
	Get(name string) (Entry, error)
	Find(term string) map[string]Entry
	List() map[string]Entry
}

//
// Registry's entry
//
type Entry struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Executable  string      `json:"exec"`
	Elementary  bool        `json:"elementary"`
	Inports     []EntryPort `json:"inports"`
	Outports    []EntryPort `json:"outports"`
}

func (entry *Entry) FindInport(name string) (EntryPort, bool) {
	for _, p := range entry.Inports {
		if p.Name == name {
			return p, true
		}
	}
	return EntryPort{}, false
}

func (entry *Entry) FindOutport(name string) (EntryPort, bool) {
	for _, p := range entry.Outports {
		if p.Name == name {
			return p, true
		}
	}
	return EntryPort{}, false
}

//
// Entry's port data
//
type EntryPort struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Addressable bool   `json:"addressable"`
}

//
// JSON Registry implements Registrar interface
//
type JSONRegistry struct {
	Name    string           `json:"name"`
	Entries map[string]Entry `json:"entries"`
	Created time.Time        `json:"created"`
	Updated time.Time        `json:"updated"`
}

func (self JSONRegistry) Add(entry Entry) {
	inports := []EntryPort{}
	outports := []EntryPort{}
	for _, p := range entry.Inports {
		p.Name = strings.ToLower(p.Name)
		inports = append(inports, p)
	}
	for _, p := range entry.Outports {
		p.Name = strings.ToLower(p.Name)
		outports = append(outports, p)
	}
	entry.Inports = inports
	entry.Outports = outports
	self.Entries[entry.Name] = entry
}

func (self JSONRegistry) Exists(name string) bool {
	_, ok := self.Entries[name]
	return ok
}

func (self JSONRegistry) Get(name string) (Entry, error) {
	if entry, ok := self.Entries[name]; ok {
		return entry, nil
	}
	return Entry{}, NotFound
}

func (self JSONRegistry) Find(term string) map[string]Entry {
	results := map[string]Entry{}
	for name, e := range self.Entries {
		if strings.Contains(name, term) {
			results[name] = e
		}
	}
	return results
}

func (self JSONRegistry) List() map[string]Entry {
	return self.Entries
}

//
// A shortcut to JSON serialization
//
func (self JSONRegistry) JSON() ([]byte, error) {
	return json.MarshalIndent(self, "", "   ")
}

//
// A shortcut for JSON serialization
func (self Entry) JSON() ([]byte, error) {
	return json.Marshal(self)
}

//
// A shortcut to parser bytes into Entry structure
func ParseEntry(data []byte) (*Entry, error) {
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}
