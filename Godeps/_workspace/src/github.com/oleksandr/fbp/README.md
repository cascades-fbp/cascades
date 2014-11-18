NoFlo's FBP DSL parser for Go
===

A Go parser for .FBP DSL language from NoFlo

Dependencies
---

This is optional. If you want to update the parser's code based on the grammer.peg you need to install the following dependency:

    go get github.com/pointlander/peg

This will download and compile _peg_ binary, which you can use later to generate the parser.

The following command will generate the parser:

    peg -switch -inline grammar.peg

Installation
---

Use regular _go install_ or _go get_ command to download and install the _fbp_ library:

    go get github.com/oleksandr/fbp

The library already includes the generated parser.

Basic usage
---

    import "github.com/oleksandr/fbp"

    var graph string = `
        '5s' -> INTERVAL Ticker(core/ticker) OUT -> IN Forward(core/passthru)
        Forward OUT -> IN Log(core/console)`

    parser := &fbp.Fbp{Buffer: graph}
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

    // At this point you have parser.Processes, parser.Connections, 
    // parser.Inports and parser.Outports data structures...




