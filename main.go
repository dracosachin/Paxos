package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Role struct {
	IsProposer bool
	IsAcceptor bool
	IsLearner  bool
}

type Message struct {
	PeerID       int    `json:"peer_id"`
	Action       string `json:"action"`
	MessageType  string `json:"message_type"`
	MessageValue string `json:"message_value"`
	ProposalNum  int    `json:"proposal_num"`
}

var (
	peerID           int
	proposalNum      int
	messageValue     string
	role             Role
	peers            map[string]Role
	acceptors        []string
	ackCount         int
	acceptAckCount   int
	ackLock          sync.Mutex
	acceptAckLock    sync.Mutex
	promisedProposal int
	acceptedProposal int
	acceptedValue    string
	preparePhaseDone bool
	acceptPhaseDone  bool
)

const (
	port               = "8080"
	initialProposalNum = 1
	prepareDelay       = 1 * time.Millisecond // Delay between sending each prepare message
	ackDelay           = 1 * time.Millisecond // Delay after receiving majority prepare_ack
)

// Function to read and parse hostsfile.txt
func readHostsFile(hostfile string) error {
	data, err := ioutil.ReadFile(hostfile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	peers = make(map[string]Role)

	for i, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		hostname := parts[0]
		roles := strings.Split(parts[1], ",")
		var r Role
		for _, role := range roles {
			switch role {
			case "proposer1":
				r.IsProposer = true
			case "acceptor1":
				r.IsAcceptor = true
				acceptors = append(acceptors, hostname)
			case "learner1":
				r.IsLearner = true
			}
		}
		peers[hostname] = r
		if hostname == os.Getenv("HOSTNAME") {
			peerID = i + 1
			role = r
		}
	}
	return nil
}

// Function to print message as JSON
func logMessage(action, msgType, msgValue string, proposalNum int) {
	msg := Message{
		PeerID:       peerID,
		Action:       action,
		MessageType:  msgType,
		MessageValue: msgValue,
		ProposalNum:  proposalNum,
	}
	jsonMsg, _ := json.Marshal(msg)
	fmt.Println(string(jsonMsg))
}

// Function to start listening for incoming messages
func startListener() {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", port))
	if err != nil {
		os.Exit(1)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

// Function for proposers to send prepare messages and wait for responses
func proposerSendPrepare() {
	for _, acceptor := range acceptors {
		if preparePhaseDone {
			break
		}
		addr := fmt.Sprintf("%s:%s", acceptor, port)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		logMessage("sent", "prepare", messageValue, proposalNum)
		prepareMsg := Message{
			PeerID:       peerID,
			Action:       "sent",
			MessageType:  "prepare",
			MessageValue: messageValue,
			ProposalNum:  proposalNum,
		}
		err = json.NewEncoder(conn).Encode(prepareMsg)
		if err != nil {
			conn.Close()
			continue
		}

		// Wait for the acceptor's response
		var resp Message
		err = json.NewDecoder(conn).Decode(&resp)
		if err != nil {
			conn.Close()
			continue
		}

		logMessage("received", resp.MessageType, resp.MessageValue, resp.ProposalNum)

		if resp.MessageType == "prepare ack" {
			handlePrepareAck(resp)
		}

		conn.Close()
		time.Sleep(prepareDelay)
	}
}

// Function for acceptors to handle prepare requests with a delay before responding
func acceptorHandlePrepare(conn net.Conn, msg Message) {
	// Simulate delay before sending prepare_ack
	time.Sleep(prepareDelay)

	if msg.ProposalNum > promisedProposal {
		promisedProposal = msg.ProposalNum // Update promised proposal number

		response := Message{
			PeerID:       peerID,
			Action:       "sent",
			MessageType:  "prepare ack",
			MessageValue: acceptedValue, // Return accepted value, if any
			ProposalNum:  acceptedProposal,
		}
		logMessage("sent", "prepare ack", response.MessageValue, response.ProposalNum)

		err := json.NewEncoder(conn).Encode(response)
		if err != nil {
			// Handle error if necessary
		}
	} else {
		// Optionally, send a rejection message
	}
}

// Function for acceptors to handle accept requests
func acceptorHandleAccept(conn net.Conn, msg Message) {
	// Simulate delay before sending accept_ack
	time.Sleep(prepareDelay)

	if msg.ProposalNum >= promisedProposal {
		promisedProposal = msg.ProposalNum
		acceptedProposal = msg.ProposalNum
		acceptedValue = msg.MessageValue

		response := Message{
			PeerID:       peerID,
			Action:       "sent",
			MessageType:  "accept ack",
			MessageValue: acceptedValue,
			ProposalNum:  acceptedProposal,
		}
		logMessage("sent", "accept ack", response.MessageValue, response.ProposalNum)

		err := json.NewEncoder(conn).Encode(response)
		if err != nil {
			// Handle error if necessary
		}
	} else {
		// Optionally, send a rejection message
	}
}

// Function to handle incoming connections and messages
func handleConnection(conn net.Conn) {
	var msg Message
	err := json.NewDecoder(conn).Decode(&msg)
	if err != nil {
		conn.Close()
		return
	}

	logMessage("received", msg.MessageType, msg.MessageValue, msg.ProposalNum)

	switch msg.MessageType {
	case "prepare":
		if role.IsAcceptor {
			acceptorHandlePrepare(conn, msg)
		}
	case "accept":
		if role.IsAcceptor {
			acceptorHandleAccept(conn, msg)
		}
	case "prepare ack":
		if role.IsProposer {
			handlePrepareAck(msg)
		}
	case "accept ack":
		if role.IsProposer {
			handleAcceptAck()
		}
	case "chose":
		handleChosenMessage(msg)
	default:
		// Unexpected message type
	}

	conn.Close()
}

// Function to handle prepare_ack messages and proceed to the accept phase if majority is reached
func handlePrepareAck(msg Message) {
	ackLock.Lock()
	defer ackLock.Unlock()
	if preparePhaseDone {
		return
	}
	ackCount++
	majority := len(acceptors)/2 + 1

	// Update the proposed value if necessary
	if msg.ProposalNum > 0 && msg.MessageValue != "" {
		// If any acceptedValues returned, replace value with acceptedValue for highest acceptedProposal
		if msg.ProposalNum > acceptedProposal {
			acceptedProposal = msg.ProposalNum
			messageValue = msg.MessageValue
		}
	}

	if ackCount >= majority {
		preparePhaseDone = true
		time.Sleep(ackDelay) // Delay to ensure ordered processing
		proposerSendAccept()
		ackCount = 0 // Reset ack count for future proposals
	}
}

// Function to handle accept_ack messages and decide if value is chosen
func handleAcceptAck() {
	acceptAckLock.Lock()
	defer acceptAckLock.Unlock()
	if acceptPhaseDone {
		return
	}
	acceptAckCount++
	majority := len(acceptors)/2 + 1

	if acceptAckCount >= majority {
		acceptPhaseDone = true
		sendChosenMessage()
		acceptAckCount = 0 // Reset for future proposals
	}
}

// Function for proposers to send accept messages after receiving majority prepare_ack
func proposerSendAccept() {
	for _, acceptor := range acceptors {
		if acceptPhaseDone {
			break
		}
		addr := fmt.Sprintf("%s:%s", acceptor, port)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		logMessage("sent", "accept", messageValue, proposalNum)
		acceptMsg := Message{
			PeerID:       peerID,
			Action:       "sent",
			MessageType:  "accept",
			MessageValue: messageValue,
			ProposalNum:  proposalNum,
		}
		err = json.NewEncoder(conn).Encode(acceptMsg)
		if err != nil {
			conn.Close()
			continue
		}

		// Wait for the acceptor's response
		var resp Message
		err = json.NewDecoder(conn).Decode(&resp)
		if err != nil {
			conn.Close()
			continue
		}

		logMessage("received", resp.MessageType, resp.MessageValue, resp.ProposalNum)

		if resp.MessageType == "accept ack" {
			handleAcceptAck()
		}

		conn.Close()
		time.Sleep(prepareDelay)
	}
}

// Function to send "chose" message to all peers
func sendChosenMessage() {
	for peer := range peers {
		addr := fmt.Sprintf("%s:%s", peer, port)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		choseMsg := Message{
			PeerID:       peerID,
			Action:       "chose",
			MessageType:  "chose",
			MessageValue: messageValue,
			ProposalNum:  proposalNum,
		}
		err = json.NewEncoder(conn).Encode(choseMsg)
		if err != nil {
			conn.Close()
			continue
		}
		conn.Close()
	}
}

// Function to handle "chose" messages
func handleChosenMessage(msg Message) {
	// Print the "chose" message as specified
	logMessage("chose", "chose", msg.MessageValue, msg.ProposalNum)
	// Optionally, update local state to reflect the chosen value
}

func main() {
	// Parse command-line flags
	hostfile := flag.String("h", "", "Path to hostsfile")
	value := flag.String("v", "", "Value to propose")
	delay := flag.Int("t", 0, "Delay before starting proposal (for proposer)")
	flag.Parse()

	if *hostfile == "" {
		os.Exit(1)
	}

	// Assign global variables
	messageValue = *value
	proposalNum = initialProposalNum // Initialize with a known value

	// Read hostsfile and set roles
	if err := readHostsFile(*hostfile); err != nil {
		os.Exit(1)
	}

	// Start the listener in a separate goroutine so it runs concurrently
	go startListener()

	// Delay if required
	if *delay > 0 {
		time.Sleep(time.Duration(*delay) * time.Second)
	}

	// Start Paxos protocol if this peer is a proposer
	if role.IsProposer {
		time.Sleep(2 * time.Second) // Wait before starting
		proposerSendPrepare()
	}

	// Keep the main function alive to allow listener to continue running
	select {}
}
