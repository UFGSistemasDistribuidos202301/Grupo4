package main

//servers addresses
const (
	REGISTER_SERVER_ADDRESS        = "localhost:8080"
	CRITICAL_REGION_SERVER_ADDRESS = "localhost:8081"
)

// types of message
const (
	REPLY = iota
	REQUEST
	PERMISSION
)

// states of process
const (
	WANTED = iota
	RELEASED
	HELD
)

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func less(p *process, msg message) bool {
	return checkProcessIsMinorThanMessage(p.requestTimestamp, p.id, msg.RequestTimestamp, msg.Id)

}

func checkProcessIsMinorThanMessage(timestampProcess, idProcess, timestampMessage, idMessage int) bool {
	if timestampProcess < timestampMessage {
		return true
	} else if (timestampProcess == timestampMessage) && (idProcess < idMessage) {
		return true
	} else {
		return false
	}
}
