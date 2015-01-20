package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cascades-fbp/cascades/graph"
	"github.com/cascades-fbp/cascades/library"
	"github.com/codegangsta/cli"
)

// Print catalog command
func listLibrary(c *cli.Context) {
	// read components library file if exists
	data, err := ioutil.ReadFile(c.GlobalString("file"))
	if err != nil && os.IsExist(err) {
		fmt.Printf("Failed to read catalogue file: %s", err.Error())
		return
	}
	if data == nil {
		fmt.Println("Library is empty")
		return
	}

	// create JSON library (the only implementation for now)
	var db library.JSONLibrary
	err = json.Unmarshal(data, &db)
	if err != nil {
		fmt.Printf("Library file is invalid: %s", err.Error())
		return
	}

	for _, e := range db.List() {
		fmt.Println(e.Name)
	}
}

func infoFromLibrary(c *cli.Context) {
	if len(c.Args()) != 1 {
		fmt.Printf("Incorrect Usage. You need to provide a component name as argument!\n\n")
		cli.ShowAppHelp(c)
		return
	}

	// read components library file if exists
	data, err := ioutil.ReadFile(c.GlobalString("file"))
	if err != nil && os.IsExist(err) {
		fmt.Printf("Failed to read catalogue file: %s", err.Error())
		return
	}
	if data == nil {
		fmt.Println("Library is empty")
		return
	}

	// create JSON library (the only implementation for now)
	var db library.JSONLibrary
	err = json.Unmarshal(data, &db)
	if err != nil {
		fmt.Printf("Library file is invalid: %s", err.Error())
		return
	}

	component := strings.ToLower(c.Args().First())
	for _, e := range db.List() {
		if e.Name != component {
			continue
		}
		fmt.Printf("NAME:\n    %s\n", e.Name)
		fmt.Printf("LOCATION:\n    %s\n", e.Executable)
		fmt.Println("INPUTS:")
		if len(e.Inports) == 0 {
			fmt.Println("    None")
		} else {
			for _, p := range e.Inports {
				fmt.Printf("    %s - type=%s, array=%v, required=%v\n", strings.ToUpper(p.Name), p.Type, p.Addressable, p.Required)
			}
		}
		fmt.Println("OUTPUTS:")
		if len(e.Outports) == 0 {
			fmt.Println("    None")
		} else {
			for _, p := range e.Outports {
				fmt.Printf("    %s - type=%s, array=%v, required=%v\n", strings.ToUpper(p.Name), p.Type, p.Addressable, p.Required)
			}

		}
		fmt.Printf("DESCRIPTION:\n    %s\n", e.Description)
		break
	}
}

// Implements catalog updating command
func addToLibrary(c *cli.Context) {
	if len(c.Args()) != 1 {
		fmt.Printf("Incorrect Usage. You need to provide a directory/file path as argument!\n\n")
		cli.ShowAppHelp(c)
		return
	}

	// read components library file if exists
	data, err := ioutil.ReadFile(c.GlobalString("file"))
	if err != nil && os.IsExist(err) {
		fmt.Printf("Failed to read catalogue file: %s", err.Error())
		return
	}

	// create JSON library (the only implementation for now)
	var db library.JSONLibrary
	if data != nil {
		err = json.Unmarshal(data, &db)
	}
	if err != nil {
		db = library.JSONLibrary{
			Entries: make(map[string]library.Entry),
		}
		db.Name = "Local Components Library"
		db.Created = time.Now()
	}

	info, err := os.Stat(c.Args().First())
	if err != nil {
		fmt.Printf("Failed to access given directory/file: %s", err.Error())
		fmt.Println("")
		return
	}

	if info.IsDir() {
		path, err := filepath.Abs(c.Args().First())
		if err != nil {
			fmt.Printf("Failed to resolve absolute path for %s: %s", c.Args().First(), err.Error())
			fmt.Println("")
			return
		}
		addDirToLibrary(c, db, path)
	} else {
		if c.String("name") == "" {
			fmt.Println("You need to provide a name when adding a component file")
			cli.ShowAppHelp(c)
			fmt.Println("")
			return
		}
		err = addFileToLibrary(c, db, c.Args().First(), c.String("name"))
		if err != nil {
			fmt.Printf("Failed to add a component: %s", err.Error())
			fmt.Println("")
			return
		}
	}

	// write index back or create if not exists
	db.Updated = time.Now()
	result, err := db.JSON()
	if err != nil {
		fmt.Printf("Failed to generate JSON: %s", err.Error())
		return
	}
	err = ioutil.WriteFile(c.GlobalString("file"), result, os.FileMode(0644))
	if err != nil {
		fmt.Printf("Failed to save registry file: %s", err.Error())
		return
	}
}

func addDirToLibrary(c *cli.Context, r library.Registrar, dir string) {
	fmt.Printf("Walking components directory: %s\n", dir)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}
		if info.IsDir() {
			return nil
		}
		sep := string(filepath.Separator)
		name := strings.TrimPrefix(strings.TrimPrefix(path, dir), sep)
		name = strings.TrimPrefix(name, sep)
		name = strings.Replace(name, sep, "/", 1)
		name = strings.TrimSuffix(name, ".fbp")
		name = strings.TrimSuffix(name, ".json")
		name = strings.TrimSuffix(name, ".exe")
		err = addFileToLibrary(c, r, path, name)
		if err != nil {
			fmt.Printf("Error adding to registry: %s", err.Error())
		}
		return nil
	})

	fmt.Println("DONE")
}

func addFileToLibrary(c *cli.Context, r library.Registrar, file string, name string) error {
	var entry *library.Entry
	if strings.HasSuffix(file, ".fbp") {
		// adding a compsite component (subgraph) in .fbp format
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		g, err := graph.ParseFBP(data)
		if err != nil {
			return err
		}
		if entry, err = graphToEntry(g, file, r); err != nil {
			return err
		}
		entry.Elementary = false

	} else if strings.HasSuffix(file, ".json") {
		// adding a compsite component (subgraph) in .json format
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		g, err := graph.ParseJSON(data)
		if err != nil {
			return err
		}
		if entry, err = graphToEntry(g, file, r); err != nil {
			return err
		}
		entry.Elementary = false

	} else {
		// adding an elementary component
		c := exec.Command(file, "--json")
		out, err := c.Output()
		if err != nil {
			return fmt.Errorf("Cannot register component %s: %s\n", name, err.Error())
		}
		if err = json.Unmarshal(out, &entry); err != nil {
			return err
		}
		entry.Elementary = true
	}

	if len(entry.Inports) == 0 && len(entry.Outports) == 0 {
		return fmt.Errorf("Cannot register component %s: inports and outports are empty", name)
	}

	entry.Name = name
	entry.Executable = file
	if r.Exists(name) && !c.Bool("force") {
		fmt.Printf("WARNING \"%s\" already exists and --force is not provided. Ignoring this entry", name)
		fmt.Println("")
	} else {
		r.Add(*entry)
		fmt.Printf("Added \"%s\"", name)
		fmt.Println("")
	}

	return nil
}

func graphToEntry(g *graph.Description, path string, r library.Registrar) (*library.Entry, error) {
	entry := &library.Entry{
		Executable:  path,
		Description: g.Properties["name"],
		Inports:     []library.EntryPort{},
		Outports:    []library.EntryPort{},
	}

	for _, e := range g.Inports {
		parts := strings.SplitN(e.Private, ".", 2)
		rec, err := r.Get(g.Processes[parts[0]].Component)
		if err != nil {
			return nil, fmt.Errorf("Component %s not found in library", parts[0])
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
			return nil, fmt.Errorf("Component %s not found in library", parts[0])
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
