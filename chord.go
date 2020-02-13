package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"time"
)

//Ross Code

//Returns a sha1 hash value
func hashString(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

const keySize = sha1.Size * 8

var two = big.NewInt(2)
var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(keySize), nil)

func jump(address string, fingerentry int) *big.Int {
	n := hashString(address)
	fingerentryminus1 := big.NewInt(int64(fingerentry) - 1)
	jump := new(big.Int).Exp(two, fingerentryminus1, nil)
	sum := new(big.Int).Add(n, jump)

	return new(big.Int).Mod(sum, hashMod)
}

func getLocalAddress() string {
	var localaddress string

	ifaces, err := net.Interfaces()
	if err != nil {
		panic("init: failed to find network interfaces")
	}

	// find the first non-loopback interface with an IP address
	for _, elt := range ifaces {
		if elt.Flags&net.FlagLoopback == 0 && elt.Flags&net.FlagUp != 0 {
			addrs, err := elt.Addrs()
			if err != nil {
				panic("init: failed to get addresses for network interface")
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ip4 := ipnet.IP.To4(); len(ip4) == net.IPv4len {
						localaddress = ip4.String()
						break
					}
				}
			}
		}
	}
	if localaddress == "" {
		panic("init: failed to find non-loopback interface with valid address on this node")
	}

	return localaddress
}

//End of Ross code

func helpCommand() {
	fmt.Println("help:              Displays a list of commands", "\n")
	fmt.Println("port <n>:          Sets the port this node should listen on", "\n")
	fmt.Println("create:            Creates a new ring if no ring has been joined or exists", "\n")
	fmt.Println("join <address>:    Joins an existing ring at the specified address", "\n")
	fmt.Println("put <key> <value>: Inserts a key/value pair into the active ring", "\n")
	fmt.Println("putrandom <n>:     Randomly generates n keys and associated values and stores them on the ring", "\n")
	fmt.Println("get <key>:         Find the given key on the ring and return its value", "\n")
	fmt.Println("delete <key>:      Deletes the given key from the ring", "\n")
	fmt.Println("dump:              Display info about current node", "\n")
	fmt.Println("quit:              Ends the program")
}

type Node struct {
	Address string
}

func create(node *Node, portNumber string) error {
	go func() {
		rpc.Register(node)
		rpc.HandleHTTP()
		log.Fatal(http.ListenAndServe(portNumber, nil), "")
	}()
	time.Sleep(time.Second)
	return nil
}

func allCommands() {
	port := ":3410"

	node := Node{
		Address: getLocalAddress() + port,
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please enter a command:")
	for scanner.Scan() {
		userCommand := strings.Split(scanner.Text(), " ")
		switch userCommand[0] {
		case "help":
			helpCommand()
		case "port":
			if len(userCommand) == 2 {
				port = userCommand[1]
				fmt.Println("Your port number: ")
			}
		case "create":
			create(&node, port)
			fmt.Println("You created a new ring")
			fmt.Println("Listening")
		case "join":
			fmt.Println("You joined a ring")
		case "put":
			fmt.Println("You put something on the ring")
		case "putrandom":
			fmt.Println("You put random crap on the ring")
		case "get":
			fmt.Println("You get something on the ring")
		case "delete":
			fmt.Println("You deleted something")
		case "dump":
			fmt.Println("Here is your node info")
		case "quit":
			fmt.Println("You quit")
			os.Exit(3)
		default:
			fmt.Println("not a valid command")
		}
	}

}

func main() {
	allCommands()

}
