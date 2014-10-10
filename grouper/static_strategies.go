package grouper

import "os"

func NewParallel(signal os.Signal, members []Member) StaticGroup {
	return newStatic(signal, members, parallelInit)
}

func parallelInit(members Members, client PoolClient) {
	insert := client.Insert()
	closed := client.CloseNotifier()

	for _, member := range members {
		select {
		case insert <- member:
		case <-closed:
			return
		}
	}
	client.Close()
	for _ = range client.NewEntranceListener() {
		// wait for all members to be ready
	}
}

func NewOrdered(signal os.Signal, members []Member) StaticGroup {
	return newStatic(signal, members, orderedInit)
}

func orderedInit(members Members, client PoolClient) {
	entranceEvents := client.NewEntranceListener()
	insert := client.Insert()
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

func serialInit(members Members, client PoolClient) {
	exitEvents := client.NewExitListener()
	insert := client.Insert()
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
