package main

const (
	RuntimeProtocolVersion = "0.4"
	RuntimeType            = "custom"
)

const (
	ProtocolRuntime   = "runtime"
	ProtocolGraph     = "graph"
	ProtocolComponent = "component"
	ProtocolNetwork   = "network"
)

type Message struct {
	ConnId   string      `json:"-"`
	Protocol string      `json:"protocol"`
	Command  string      `json:"command"`
	Payload  interface{} `json:"payload"`
}

type Capability string

const (
	// The runtime is able to expose the ports of its main graph using the Runtime protocol and transmit packet information to/from them
	CapabilityProtocolRuntime Capability = "protocol:runtime"

	// The runtime is able to modify its graphs using the Graph protocol
	CapabilityProtocolGraph Capability = "protocol:graph"

	// The runtime is able to list and modify its components using the Component protocol
	CapabilityProtocolComponent Capability = "protocol:component"

	// The runtime is able to control and introspect its running networks using the Network protocol
	CapabilityProtocolNetwork Capability = "protocol:network"

	// The runtime is able to compile and run custom components sent as source code strings
	CapabilityProtocolSetSource = "component:setsource"

	// The runtime is able to read and send component source code back to client
	CapabilityProtocolGetSource = "component:getsource"

	// The runtime is able to "flash" a running graph setup into itself, making it persistent across reboots
	CapabilityNetworkPersist Capability = "network:persist"
)

type RuntimePayload struct {
	Type         string       `json:"type"`            // type of the runtime, for example noflo-nodejs or microflo
	Version      string       `json:"version"`         // version of the runtime protocol that the runtime supports, for example 0.4
	Capabilities []Capability `json:"capabilities"`    // array of capability strings for things the runtime is able to do
	Id           string       `json:"id,omitempty"`    // (optional) runtime ID used with Flowhub Registry
	Label        string       `json:"label,omitempty"` // (optional) Human-readable description of the runtime
	Graph        string       `json:"graph,omitempty"` // (optional) ID of the currently configured main graph running on the runtime, if any
}

type ComponentPayload struct {
	Name        string                    `json:"name"`                  // component name in format that can be used in graphs
	Description string                    `json:"description,omitempty"` // (optional) textual description on what the component does
	Icon        string                    `json:"icon,omitempty"`        // (optional) matching icon names http://fortawesome.github.io/Font-Awesome/icons/
	Subgraph    bool                      `json:"subgraph"`              // telling whether the component is a subgraph
	Inports     []ComponentPayloadInport  `json:"inPorts"`               // list of input ports
	Outports    []ComponentPayloadOutport `json:"outPorts"`              // list of output ports
}

type ComponentPayloadInport struct {
	Id          string   `json:"id"`                    // port name
	Type        string   `json:"type"`                  // port datatype, for example boolean
	Description string   `json:"description,omitempty"` // (optional) port description
	Addressable bool     `json:"addressable"`           // telling whether the port is an ArrayPort
	Required    bool     `json:"required"`              // telling whether the port needs to be connected for the component to work
	Values      []string `json:"values,omitempty"`      // array of the values that the port accepts for enum ports
	Default     string   `json:"default,omitempty"`     // (optional) the default value received by the port
}

type ComponentPayloadOutport struct {
	Id          string `json:"id"`                    // port name
	Type        string `json:"type"`                  // port datatype, for example boolean
	Description string `json:"description,omitempty"` // (optional) port description
	Addressable bool   `json:"addressable"`           // telling whether the port is an ArrayPort
	Required    bool   `json:"required"`              // telling whether the port needs to be connected for the component to work
}
