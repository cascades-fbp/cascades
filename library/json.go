package library

import (
	"encoding/json"
	"strings"
	"time"
)

//
// JSONLibrary implements Registrar interface
//
type JSONLibrary struct {
	Name    string           `json:"name"`
	Entries map[string]Entry `json:"entries"`
	Created time.Time        `json:"created"`
	Updated time.Time        `json:"updated"`
}

// Add a new entry to library
func (l JSONLibrary) Add(entry Entry) {
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
	l.Entries[entry.Name] = entry
}

// Exists returns true if an entry with a given name already exists
func (l JSONLibrary) Exists(name string) bool {
	_, ok := l.Entries[name]
	return ok
}

// Get returns an entry by given name
func (l JSONLibrary) Get(name string) (Entry, error) {
	if entry, ok := l.Entries[name]; ok {
		return entry, nil
	}
	return Entry{}, ErrNotFound
}

// Find returns a map of entries which name contains a given term
func (l JSONLibrary) Find(term string) map[string]Entry {
	results := map[string]Entry{}
	for name, e := range l.Entries {
		if strings.Contains(name, term) {
			results[name] = e
		}
	}
	return results
}

// List returns a complete entries map
func (l JSONLibrary) List() map[string]Entry {
	return l.Entries
}

// JSON is a shortcut to JSON serialization
func (l JSONLibrary) JSON() ([]byte, error) {
	return json.MarshalIndent(l, "", "   ")
}

// JSON is a shortcut for JSON serialization
func (e Entry) JSON() ([]byte, error) {
	return json.Marshal(e)
}
