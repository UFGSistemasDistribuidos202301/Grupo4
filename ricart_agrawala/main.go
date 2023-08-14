package main

func main() {
	const NUMBER_PROCESSES = 3
	const INITIAL_PORT = 8090

	var p [NUMBER_PROCESSES]process

	waiter := make(chan bool)

	for i := 0; i < NUMBER_PROCESSES; i++ {
		port := INITIAL_PORT + i
		p[i] = newProcess(port)

		go func(i int) {
			p[i].startProcess()
			waiter <- true
		}(i)
	}

	for i := 0; i < NUMBER_PROCESSES; i++ {
		<-waiter
	}

	for i := 0; i < NUMBER_PROCESSES; i++ {
		go p[i].runRicartAgrawala()
	}

	<-waiter
}
