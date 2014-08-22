package runtime

const (
	IPTypePacket       byte = 0x00
	IPTypeOpenBracket  byte = 0x01
	IPTypeCloseBracket byte = 0x02
)

func NewPacket(payload []byte) [][]byte {
	return [][]byte{[]byte{IPTypePacket}, payload}
}

func NewOpenBracket() [][]byte {
	return [][]byte{[]byte{IPTypeOpenBracket}, []byte{}}
}

func NewCloseBracket() [][]byte {
	return [][]byte{[]byte{IPTypeCloseBracket}, []byte{}}
}

func IsValidIP(ip [][]byte) bool {
	return len(ip) == 2 && len(ip[0]) == 1
}

func IsPacket(ip [][]byte) bool {
	if len(ip[0]) == 0 {
		return false
	}
	return ip[0][0] == IPTypePacket
}

func IsOpenBracket(ip [][]byte) bool {
	if len(ip[0]) == 0 {
		return false
	}
	return ip[0][0] == IPTypeOpenBracket
}

func IsCloseBracket(ip [][]byte) bool {
	if len(ip[0]) == 0 {
		return false
	}
	return ip[0][0] == IPTypeCloseBracket
}
