package library

import (
	"errors"
)

var (
	// ErrNotFound describes a case when a component not found
	ErrNotFound = errors.New("Component not found")
)

//
// Registrar defines the interface of components library
//
type Registrar interface {
	Add(entry Entry)
	Exists(name string) bool
	Get(name string) (Entry, error)
	Find(term string) map[string]Entry
	List() map[string]Entry
}

//
// Entry of a registry
//
type Entry struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Executable  string      `json:"exec"`
	Elementary  bool        `json:"elementary"`
	Inports     []EntryPort `json:"inports"`
	Outports    []EntryPort `json:"outports"`
}

// FindInport looks for an input port by name
func (entry *Entry) FindInport(name string) (EntryPort, bool) {
	for _, p := range entry.Inports {
		if p.Name == name {
			return p, true
		}
	}
	return EntryPort{}, false
}

// FindOutport loos for an output port by name
func (entry *Entry) FindOutport(name string) (EntryPort, bool) {
	for _, p := range entry.Outports {
		if p.Name == name {
			return p, true
		}
	}
	return EntryPort{}, false
}

// EntryPort represents entry's port description
type EntryPort struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Addressable bool   `json:"addressable"`
}
