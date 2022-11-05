package main

type server struct {
	port    int
	address string
}

func newServer() server {
	var tmp server
	return tmp
}

// Running the server accepts incomming connections
// Out going connections are mannaged by the appropreate repository runner
func (serv *server) Run() {

	select {}
}
