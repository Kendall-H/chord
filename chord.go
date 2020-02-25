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
	"strconv"
	"strings"
	"time"
)

//Returns a sha1 hash value
func hashString(n string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(n))
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

func between(start, n, end *big.Int, inclusive bool) bool {
	if end.Cmp(start) > 0 {
		return (start.Cmp(n) < 0 && n.Cmp(end) < 0) || (inclusive && n.Cmp(end) == 0)
	} else {
		return start.Cmp(n) < 0 || n.Cmp(end) < 0 || (inclusive && n.Cmp(end) == 0)
	}
}

func getLocalAddress() string {
	var localaddress string

	ifaces, err := net.Interfaces()
	if err != nil {
		panic("init: failed to find network interfaces")
	}

	// find the first non-loopback interface with an IP address
	for _, n := range ifaces {
		if n.Flags&net.FlagLoopback == 0 && n.Flags&net.FlagUp != 0 {
			address, err := n.Addrs()
			if err != nil {
				panic("init: failed to get addresses for network interface")
			}

			for _, addr := range address {
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

func call(address string, method string, request interface{}, reply interface{}) error {
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		log.Printf("rpc.DialHTTP: %v", err)
		return err
	}
	defer client.Close()
	return client.Call("Node."+method, request, reply)
}

func (n *Node) Ping(address string, pingBool *bool) error {
	*pingBool = true
	return nil
}

func (n *Node) Join(address string, successor *string) error {
	*successor = n.find(address)
	return nil
}

func (n *Node) Put(keyvalue *KeyValue, empty *struct{}) error {
	n.Bucket[keyvalue.Key] = keyvalue.Value
	log.Printf("\t%s was added to this node", *keyvalue)
	return nil
}

func (n *Node) Get(key string, value *string) error {
	fmt.Println("Got in get")
	if val, ok := n.Bucket[key]; ok {
		*value = val
		log.Printf("\t{%s %s} value was retrieved from this node", key, val)
		return nil
	}
	return fmt.Errorf("\tKey '%s' does not exist in ring", key)
}

func (n *Node) Delete(keyvalue *KeyValue, empty *struct{}) error {
	if value, ok := n.Bucket[keyvalue.Key]; ok {
		delete(n.Bucket, keyvalue.Key)
		log.Printf("\t{%s %s} was removed from this node", keyvalue.Key, value)
		return nil
	}
	return fmt.Errorf("\tKey '%s' does not exist in ring", keyvalue.Key)
}

func (n *Node) Dump(empty1 *struct{}, dumpN *Node) error {
	dumpN.Address = n.Address
	dumpN.Predecessor = n.Predecessor
	dumpN.Successors = n.Successors
	dumpN.Bucket = n.Bucket
	var old string
	for i := 0; i < len(n.FingerTable); i++ {
		if old != n.FingerTable[i] {
			dumpN.FingerTable = append(dumpN.FingerTable, strconv.Itoa(i)+":\t", n.FingerTable[i], "\n\t\t\t")
			old = n.FingerTable[i]
		}
	}
	return nil
}

func (n *Node) stabilize() error {
	//println("Stabilize")
	var successors []string
	//fmt.Println("successor[0] " + n.Successors[0])
	err := call(n.Successors[0], "GetSuccessors", &struct{}{}, &successors)
	//fmt.Println(&successors)
	if err == nil {
		// fmt.Println("got here")
		// fmt.Println(n.Successors[0])
		// fmt.Println(n.Successors[1])

		n.Successors[1] = successors[0]
		// fmt.Println(n.Successors[0])
		// fmt.Println(n.Successors[1])
		n.Successors[2] = successors[1]
	} else {
		log.Printf("\tPrimary successor '%s' failed", n.Successors[0])
		if n.Successors[0] == "" {
			log.Printf("\tSetting primary successor to address of this node '%s'", n.Address)
			n.Successors[0] = n.Address
		} else {
			log.Printf("\tSetting secondary successor '%s' as primary ", n.Successors[1])
			n.Successors[0] = n.Successors[1]
			n.Successors[1] = n.Successors[2]
			n.Successors[2] = ""
		}
	}
	x := ""
	call(n.Successors[0], "GetPredecessor", &struct{}{}, &x)

	if between(hashString(n.Address),
		hashString(x),
		hashString(n.Successors[0]),
		false) && x != "" {
		log.Printf("\tSetting primary successor to '%s'", x)
		n.Successors[0] = x
	}

	err = call(n.Successors[0], "Notify", n.Address, &struct{}{})
	if err != nil {
	}
	return nil
}

func (n *Node) GetPredecessor(empty1 *struct{}, predecessor *string) error {
	*predecessor = n.Predecessor
	return nil
}

func (n *Node) GetSuccessors(none *struct{}, successors *[]string) error {
	//fmt.Println("In get successors function")
	*successors = n.Successors[:]
	//fmt.Println("succesor list: ", n.Successors[:])
	return nil
}

func (n *Node) Notify(address string, empty *struct{}) error {
	if n.Predecessor == "" ||
		between(hashString(n.Predecessor), hashString(address), hashString(n.Address), false) {
		n.Predecessor = address
	}
	return nil
}

func (n *Node) closestPreceedingNode(id *big.Int) string {
	for i := len(n.FingerTable) - 1; i > 0; i-- {
		if between(hashString(n.Address), hashString(n.FingerTable[i]), id, false) {
			return n.FingerTable[i]
		}
	}
	return n.Successors[0]
}

func (n *Node) FindSuccessor(hash *big.Int, nextNode *NextNode) error {
	if between(hashString(n.Address), hash, hashString(n.Successors[0]), true) {
		nextNode.Address = n.Successors[0]
		nextNode.Found = true
		return nil
	}
	nextNode.Address = n.closestPreceedingNode(hash)
	return nil
}

func (n *Node) find(key string) string {
	nextNode := NextNode{
		Address: "",
		Found:   false,
	}
	nextNode.Address = n.Successors[0]
	count := 32
	for !nextNode.Found {
		if count > 0 {
			//log.Printf("find is calling FindSuccessor")
			err := call(nextNode.Address, "FindSuccessor", hashString(key), &nextNode)
			if err == nil {
				count--
			} else {
				count = 0
			}
		} else {
			return ""
		}
	}
	return nextNode.Address
}

func (n *Node) fixFingers() error {
	//println("running fixFin")
	n.Next++
	if n.Next > len(n.FingerTable)-1 {
		n.Next = 0
	}
	bigInt := jump(n.Address, n.Next)
	bigString := bigInt.String()

	address := n.find(bigString)

	if n.FingerTable[n.Next] != address && address != "" {
		log.Printf("\tWriting FingerTable entry '%d' as '%s'\n", n.Next, address)
		n.FingerTable[n.Next] = address
	}
	for {
		n.Next++
		if n.Next > len(n.FingerTable)-1 {
			n.Next = 0
			return nil
		}

		if between(hashString(n.Address), jump(n.Address, n.Next), hashString(address), false) && address != "" {
			n.FingerTable[n.Next] = address
		} else {
			n.Next--
			return nil
		}
	}
}

func (n *Node) checkPredecessor() error {
	//log.Println("Running checkPred")
	if n.Predecessor != "" {
		client, err := rpc.DialHTTP("tcp", n.Predecessor)
		if err != nil {
			log.Printf("\tPredecessor '%s' has failed", n.Predecessor)
			n.Predecessor = ""
			//n.Successors[0] = n.Address
		} else {
			client.Close()
		}
	}
	return nil
}

func helpCommand() {
	fmt.Println("help:              Displays a list of commands")
	fmt.Println("port <n>:          Sets the port this node should listen on")
	fmt.Println("create:            Creates a new ring if no ring has been joined or exists")
	fmt.Println("join <address>:    Joins an existing ring at the specified address")
	fmt.Println("put <key> <value>: Inserts a key/value pair into the active ring")
	fmt.Println("putrandom <n>:     Randomly generates n keys and associated thisDumpNode and stores them on the ring")
	fmt.Println("get <key>:         Find the given key on the ring and return its value")
	fmt.Println("delete <key>:      Deletes the given key from the ring")
	fmt.Println("dump:              Display dumpN about current node")
	fmt.Println("quit:              Ends the program")
}

type Node struct {
	Address     string
	Predecessor string
	Successors  [3]string
	Bucket      map[string]string
	FingerTable []string
	Next        int
}

type KeyValue struct {
	Key   string
	Value string
}

type NextNode struct {
	Address string
	Found   bool
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

}

func main() {
	//allCommands()
	existingRing := false

	port := ":3410"

	node := Node{
		Address:     getLocalAddress() + port,
		Predecessor: "",
		Successors:  [3]string{getLocalAddress() + port},
		Bucket:      make(map[string]string),
		FingerTable: make([]string, 161),
		Next:        0,
	}

	go func() {
		for {
			if existingRing {
				node.checkPredecessor()
				node.stabilize()
				node.fixFingers()
			}
			time.Sleep(time.Second)

		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please enter a command fool:")
	for scanner.Scan() {
		userCommand := strings.Split(scanner.Text(), " ")
		switch userCommand[0] {
		case "help":
			helpCommand()
		case "port":
			if len(userCommand) == 2 {
				port = ":" + userCommand[1]
				fmt.Println("Your port number: ", port)
				node.Address = getLocalAddress() + port
			}
		case "create":
			if existingRing == false {
				fmt.Println(port)
				create(&node, port)
				existingRing = true
				fmt.Println("You created a new ring")
				fmt.Println("Listening on", node.Address)
			} else {
				println("Ring already exists")
			}
		case "join":
			if existingRing == false {
				if len(userCommand) == 2 {
					create(&node, port)
					existingRing = true
					var successor string
					log.Printf("This is the node.Address " + node.Address)
					err := call(userCommand[1], "Join", node.Address, &successor)
					if err == nil {
						// log.Printf("Port #:" + port)
						// log.Printf("Successor set to" + successor)
						// log.Printf("This is your address: " + node.Address)
						node.Successors[0] = successor
					} else {
						fmt.Println(err)
					}
				}
			}
		case "ping":
			if existingRing == true {
				if len(userCommand) == 2 {
					pingBool := false
					call(node.Address, "Ping", userCommand[1], &pingBool)
					if pingBool == true {
						fmt.Printf("successfully pinged '%s'", userCommand[1])
					} else {
						fmt.Printf("Not work")
					}
				} else {
					fmt.Printf("Invalide command length")
				}
			} else {
				fmt.Printf("No ring")
			}
		case "put":
			if existingRing == true {
				if len(userCommand) == 3 {
					keyval := KeyValue{userCommand[1], userCommand[2]}
					call(node.find(userCommand[1]), "Put", keyval, &struct{}{})
					fmt.Println("You put", keyval.Key, "on the ring")
				} else {
					fmt.Printf("no work")
				}
			}
		case "putrandom":
			fmt.Println("You put random crap on the ring")
		case "get":
			if existingRing == true {
				if len(userCommand) == 2 {
					item := KeyValue{userCommand[1], ""}
					// fmt.Println("Right before the call")
					call(node.find(userCommand[1]), "Get", item.Key, &item.Value)
					// fmt.Println(item)
					fmt.Println("The value at" + "'" + item.Key + "'" + "is" + "'" + item.Value + "'")
				} else {
					fmt.Println("This didnt work")
				}
			}
		case "delete":
			if existingRing == true {
				if len(userCommand) == 3 {
					keyval := KeyValue{userCommand[2], ""}
					call(userCommand[1], "Delete", keyval, node)
				}
			}
		case "dump":
			if existingRing == true {
				fmt.Println("This is working")
				fmt.Println("Node address: " + node.Address)
				if len(userCommand) == 1 {
					var thisDumpNode Node
					err := call(node.Address, "Dump", &struct{}{}, &thisDumpNode)
					if err == nil {
						fmt.Println("Address: 		", thisDumpNode.Address)
						fmt.Println("Pred: 			", thisDumpNode.Predecessor)
						fmt.Println("Successors:	", thisDumpNode.Successors)
						fmt.Println("Bucket: 		", thisDumpNode.Bucket)
						fmt.Println("FingerTable: 	", thisDumpNode.FingerTable)
					}
				}
			}
		case "quit":
			fmt.Println("You quit")
			os.Exit(3)
		default:
			fmt.Println("not a valid command")
		}
	}
}
