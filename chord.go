package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"math/big"
	"os"
	"strings"
)

//Returns a sha1 hash value
func hashString(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

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

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please enter a command:")
	for scanner.Scan() {
		userCommand := strings.Split(scanner.Text(), " ")
		switch userCommand[0] {
		case "help":
			helpCommand()
		case "port":
			fmt.Println("Your port number")
		case "create":
			fmt.Println("You created a new ring")
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
		}
	}
}
