package registry

import (
	"encoding/json"
	"strings"
	"time"
)

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
