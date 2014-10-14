package grouper

import "os"

func NewParallel(signal os.Signal, members []Member) StaticGroup {
	return newStatic(signal, members, parallelInit)
}

func parallelInit(members Members, client DynamicClient) {
	insert := client.Inserter()
	closed := client.CloseNotifier()

	for _, member := range members {
		select {
		case insert <- member:
		case <-closed:
			return
		}
	}
	client.Close()
	for _ = range client.EntranceListener() {
		// wait for all members to be ready
	}
}

func NewOrdered(signal os.Signal, members []Member) StaticGroup {
	return newStatic(signal, members, orderedInit)
}

func orderedInit(members Members, client DynamicClient) {
	entranceEvents := client.EntranceListener()
	insert := client.Inserter()
	closed := client.CloseNotifier()

	for _, member := range members {
		select {
		case insert <- member:
		case <-closed:
			return
		}
		<-entranceEvents
	}
}

func NewSerial(members []Member) StaticGroup {
	return newStatic(nil, members, serialInit)
}

func serialInit(members Members, client DynamicClient) {
	exitEvents := client.ExitListener()
	insert := client.Inserter()
	closed := client.CloseNotifier()

	for _, member := range members {
		select {
		case insert <- member:
		case <-closed:
			return
		}

		exit := <-exitEvents
		if exit.Err != nil {
			return
		}
	}
}
