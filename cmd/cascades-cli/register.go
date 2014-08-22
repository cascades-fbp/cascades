package main

import (
	"cascades/graph"
	"cascades/log"
	"cascades/registry"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Root account command
var cmdRegister = &cobra.Command{
	Use:   "register",
	Short: "Add directory of components or separate components to registry.",
	Long:  "Add directory of components or separate components to registry.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var cmdRegisterFile = &cobra.Command{
	Use:   "file [path]",
	Short: "Adds a given file as a componnet to a registry",
	Run:   cmdRegisterFileFunc,
}

var cmdRegisterDir = &cobra.Command{
	Use:   "dir [path]",
	Short: "Walks the given directory and register every found executable as component",
	Run:   cmdRegisterDirFunc,
}

func cmdRegisterFileFunc(cmd *cobra.Command, args []string) {
	// check input params
	if len(args) != 1 {
		cmd.Usage()
		return
	}
	componentFile := args[0]
	name := cmd.Flag("name").Value.String()
	if name == "" {
		cmd.Usage()
		return
	}

	if indexFilepath == "" {
		cmd.Usage()
		return
	}

	// parse index if exists
	data, err := ioutil.ReadFile(indexFilepath)
	if err != nil && os.IsExist(err) {
		log.ErrorOutput("Failed to read existing index file: " + err.Error())
		return
	}
	var db registry.JSONRegistry
	if data != nil {
		err = json.Unmarshal(data, &db)
	}
	if err != nil {
		db = registry.JSONRegistry{
			Entries: make(map[string]registry.Entry),
		}
		db.Name = "Local Components Registry"
		db.Created = time.Now()
	}

	// add component data to registry
	err = addComponentToRegistry(cmd, name, componentFile, &db)
	if err != nil {
		log.ErrorOutput("ERROR adding to registry: " + err.Error())
		return
	}

	// write index back or create if not exists
	db.Updated = time.Now()
	result, err := db.JSON()
	if err != nil {
		log.ErrorOutput("Failed to generate JSON: " + err.Error())
		return
	}
	err = ioutil.WriteFile(indexFilepath, result, os.FileMode(0644))
	if err != nil {
		log.ErrorOutput("Failed to write index file: " + err.Error())
		return
	}

	cmd.Println("Done!")
}

func cmdRegisterDirFunc(cmd *cobra.Command, args []string) {
	// check input params
	if len(args) != 1 {
		cmd.Usage()
		return
	}
	componentsDir := args[0]
	if indexFilepath == "" {
		cmd.Usage()
		return
	}

	// parse index if exists
	data, err := ioutil.ReadFile(indexFilepath)
	if err != nil && os.IsExist(err) {
		log.ErrorOutput("Failed to read existing index file: " + err.Error())
		return
	}
	var db registry.JSONRegistry
	if data != nil {
		err = json.Unmarshal(data, &db)
	}
	if err != nil {
		db = registry.JSONRegistry{
			Entries: make(map[string]registry.Entry),
		}
		db.Name = "Local Components Registry"
		db.Created = time.Now()
	}

	// index the components
	log.SystemOutput("Walking components directory: " + componentsDir)
	filepath.Walk(componentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			cmd.Println(err.Error())
			return nil
		}
		if info.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(strings.TrimPrefix(path, componentsDir), "/")
		name = strings.TrimSuffix(name, ".fbp")
		name = strings.TrimSuffix(name, ".json")
		name = strings.TrimSuffix(name, ".exe")
		err = addComponentToRegistry(cmd, name, path, &db)
		if err != nil {
			log.ErrorOutput("adding to registry: " + err.Error())
		}
		return nil
	})

	// write index back or create if not exists
	db.Updated = time.Now()
	result, err := db.JSON()
	if err != nil {
		log.ErrorOutput("Failed to generate JSON: " + err.Error())
		return
	}
	err = ioutil.WriteFile(indexFilepath, result, os.FileMode(0644))
	if err != nil {
		log.ErrorOutput("Failed to write index file: " + err.Error())
		return
	}

	log.SystemOutput("Done!")
}

func addComponentToRegistry(cmd *cobra.Command, name, path string, r *registry.JSONRegistry) error {
	var entry *registry.Entry
	if strings.HasSuffix(path, ".fbp") {
		// adding a compsite component (subgraph) in .fbp format
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		g, err := graph.ParseFBP(data)
		if err != nil {
			return err
		}
		if entry, err = graphToEntry(g, path, r); err != nil {
			return err
		}
		entry.Elementary = false

	} else if strings.HasSuffix(path, ".json") {
		// adding a compsite component (subgraph) in .json format
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		g, err := graph.ParseJSON(data)
		if err != nil {
			return err
		}
		if entry, err = graphToEntry(g, path, r); err != nil {
			return err
		}
		entry.Elementary = false

	} else {
		// adding an elementary component
		c := exec.Command(path, "-json")
		out, err := c.Output()
		if err != nil {
			return fmt.Errorf("Cannot register component %s: %s", name, err.Error())
		}
		entry, err = registry.ParseEntry(out)
		if err != nil {
			return err
		}
		entry.Elementary = true
	}

	if len(entry.Inports) == 0 && len(entry.Outports) == 0 {
		return fmt.Errorf("Cannot register component %s: inports and outports are empty", name)
	}

	entry.Name = name
	entry.Executable = path
	if r.Exists(name) && !forceCommand {
		cmd.Println("WARNING component", name, "already exists and --force is not provided. Ignoring this entry")
	} else {
		r.Add(*entry)
	}
	return nil
}

func graphToEntry(g *graph.GraphDescription, path string, r *registry.JSONRegistry) (*registry.Entry, error) {
	entry := &registry.Entry{
		Executable:  path,
		Description: g.Properties["name"],
		Inports:     []registry.EntryPort{},
		Outports:    []registry.EntryPort{},
	}

	for _, e := range g.Inports {
		parts := strings.SplitN(e.Private, ".", 2)
		rec, err := r.Get(g.Processes[parts[0]].Component)
		if err != nil {
			return nil, fmt.Errorf("Component %s not found in registry", parts[0])
		}
		port, found := rec.FindInport(strings.ToLower(parts[1]))
		if !found {
			return nil, fmt.Errorf("Port %s not found in component %s", parts[1], parts[0])
		}
		port.Name = e.Public
		entry.Inports = append(entry.Inports, port)
	}

	for _, e := range g.Outports {
		parts := strings.SplitN(e.Private, ".", 2)
		rec, err := r.Get(g.Processes[parts[0]].Component)
		if err != nil {
			return nil, fmt.Errorf("Component %s not found in registry", parts[0])
		}
		port, found := rec.FindOutport(strings.ToLower(parts[1]))
		if !found {
			return nil, fmt.Errorf("Port %s not found in component %s", parts[1], parts[0])
		}
		port.Name = e.Public
		entry.Outports = append(entry.Outports, port)
	}

	return entry, nil
}
