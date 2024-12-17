package src

import (
	"os"
)

type Contact struct {
	Name      string
	Address   string
	Offerings []string
}

type Directory struct {
	Contacts []Contact
}

var ICA = Contact{
	Name:      "ICA",
	Address:   "server1:50051",
	Offerings: []string{"Process"},
}

var Willys = Contact{
	Name:      "Willys",
	Address:   "server2:50052",
	Offerings: []string{"Process", "Product"},
}

var Coop = Contact{
	Name:      "Coop",
	Address:   "server3:50053",
	Offerings: []string{"Product"},
}

var MyDir = Directory{
	Contacts: []Contact{ICA, Willys, Coop},
}

var Port = os.Getenv("GRPC_PORT")
