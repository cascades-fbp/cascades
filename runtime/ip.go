package runtime

const (
	// IPTypePacket represent 'data' type of IP
	IPTypePacket byte = 0x00
	// IPTypeOpenBracket represents '[' type of IP
	IPTypeOpenBracket byte = 0x01
	// IPTypeCloseBracket represents ']' type of IP
	IPTypeCloseBracket byte = 0x02
)

// NewPacket is a 'data' IP constructor
func NewPacket(payload []byte) [][]byte {
	return [][]byte{[]byte{IPTypePacket}, payload}
}

// NewOpenBracket is a '[' IP constructor
func NewOpenBracket() [][]byte {
	return [][]byte{[]byte{IPTypeOpenBracket}, []byte{}}
}

// NewCloseBracket is a ']' IP constructor
func NewCloseBracket() [][]byte {
	return [][]byte{[]byte{IPTypeCloseBracket}, []byte{}}
}

// IsValidIP checks if the given IP contains all required parts (valid)
func IsValidIP(ip [][]byte) bool {
	return len(ip) == 2 && len(ip[0]) == 1
}

// IsPacket checks if a given IP is 'data' IP
func IsPacket(ip [][]byte) bool {
	if len(ip[0]) == 0 {
		return false
	}
	return ip[0][0] == IPTypePacket
}

// IsOpenBracket checks if a given IP is '[' IP
func IsOpenBracket(ip [][]byte) bool {
	if len(ip[0]) == 0 {
		return false
	}
	return ip[0][0] == IPTypeOpenBracket
}

// IsCloseBracket checks if a given IP is ']' IP
func IsCloseBracket(ip [][]byte) bool {
	if len(ip[0]) == 0 {
		return false
	}
	return ip[0][0] == IPTypeCloseBracket
}
