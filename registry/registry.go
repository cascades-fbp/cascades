package registry

import (
	"errors"
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
