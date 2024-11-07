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
	IsProposer    bool
	ProposerID    int
	AcceptorRoles []string
	IsAcceptor    bool
	IsLearner     bool
}

type Message struct {
	PeerID       int    `json:"peer_id"`
	Action       string `json:"action"`
	MessageType  string `json:"message_type"`
	MessageValue string `json:"message_value"`
	ProposalNum  int    `json:"proposal_num"`
	AcceptorRole string // Internal use; not included in logs
}

var (
	peerID           int
	proposalNum      int
	initialValue     string
	proposedValue    string
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
	round            int
	proposerRole     string
	prepareAcks      []Message
)

const (
	port         = "8080"
	prepareDelay = 100 * time.Millisecond
	ackDelay     = 500 * time.Millisecond
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
		for _, roleStr := range roles {
			switch roleStr {
			case "proposer1":
				r.IsProposer = true
				r.ProposerID = 1
				proposerRole = "acceptor1"
			case "proposer2":
				r.IsProposer = true
				r.ProposerID = 2
				proposerRole = "acceptor2"
			case "acceptor1":
				r.IsAcceptor = true
				r.AcceptorRoles = append(r.AcceptorRoles, "acceptor1")
				if !contains(acceptors, hostname) {
					acceptors = append(acceptors, hostname)
				}
			case "acceptor2":
				r.IsAcceptor = true
				r.AcceptorRoles = append(r.AcceptorRoles, "acceptor2")
				if !contains(acceptors, hostname) {
					acceptors = append(acceptors, hostname)
				}
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

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// Function to print message as JSON without the AcceptorRole field
func logMessage(action, msgType, msgValue string, proposalNum int) {
	logMsg := struct {
		PeerID       int    `json:"peer_id"`
		Action       string `json:"action"`
		MessageType  string `json:"message_type"`
		MessageValue string `json:"message_value"`
		ProposalNum  int    `json:"proposal_num"`
	}{
		PeerID:       peerID,
		Action:       action,
		MessageType:  msgType,
		MessageValue: msgValue,
		ProposalNum:  proposalNum,
	}
	jsonMsg, _ := json.Marshal(logMsg)
	fmt.Println(string(jsonMsg))
}

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

func proposerSendPrepare() {
	// Generate a unique proposal number
	round++
	proposalNum = (round * 10) + role.ProposerID
	proposedValue = initialValue

	for _, acceptor := range acceptors {
		if preparePhaseDone {
			break
		}
		// Check if acceptor has matching acceptor role
		acceptorRoles := peers[acceptor].AcceptorRoles
		if !contains(acceptorRoles, proposerRole) {
			continue
		}

		addr := fmt.Sprintf("%s:%s", acceptor, port)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		logMessage("sent", "prepare", proposedValue, proposalNum)
		prepareMsg := Message{
			PeerID:       peerID,
			Action:       "sent",
			MessageType:  "prepare",
			MessageValue: proposedValue,
			ProposalNum:  proposalNum,
			AcceptorRole: proposerRole,
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
	}
}

func acceptorHandlePrepare(conn net.Conn, msg Message) {
	if msg.ProposalNum > promisedProposal {
		promisedProposal = msg.ProposalNum // Update promised proposal number

		response := Message{
			PeerID:       peerID,
			Action:       "sent",
			MessageType:  "prepare ack",
			MessageValue: acceptedValue,    // Return accepted value, if any
			ProposalNum:  acceptedProposal, // Return accepted proposal number
			AcceptorRole: msg.AcceptorRole,
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

func acceptorHandleAccept(conn net.Conn, msg Message) {
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
			AcceptorRole: msg.AcceptorRole,
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
		if role.IsAcceptor && contains(role.AcceptorRoles, msg.AcceptorRole) {
			acceptorHandlePrepare(conn, msg)
		}
	case "accept":
		if role.IsAcceptor && contains(role.AcceptorRoles, msg.AcceptorRole) {
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

func handlePrepareAck(msg Message) {
	ackLock.Lock()
	defer ackLock.Unlock()
	if preparePhaseDone {
		return
	}
	ackCount++
	prepareAcks = append(prepareAcks, msg)
	majority := len(acceptors)/2 + 1

	if ackCount >= majority {
		preparePhaseDone = true
		determineProposedValue()
		proposerSendAccept()
		ackCount = 0 // Reset ack count for future proposals
	}
}

func determineProposedValue() {
	highestProposalNum := -1
	value := initialValue
	for _, ack := range prepareAcks {
		if ack.ProposalNum > highestProposalNum && ack.MessageValue != "" {
			highestProposalNum = ack.ProposalNum
			value = ack.MessageValue
		}
	}
	proposedValue = value
}

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

func proposerSendAccept() {
	for _, acceptor := range acceptors {
		if acceptPhaseDone {
			break
		}
		// Check if acceptor has matching acceptor role
		acceptorRoles := peers[acceptor].AcceptorRoles
		if !contains(acceptorRoles, proposerRole) {
			continue
		}

		addr := fmt.Sprintf("%s:%s", acceptor, port)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		logMessage("sent", "accept", proposedValue, proposalNum)
		acceptMsg := Message{
			PeerID:       peerID,
			Action:       "sent",
			MessageType:  "accept",
			MessageValue: proposedValue,
			ProposalNum:  proposalNum,
			AcceptorRole: proposerRole,
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
	}
}

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
			MessageValue: proposedValue,
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

func handleChosenMessage(msg Message) {
	// Print the "chose" message as specified
	logMessage("chose", "chose", msg.MessageValue, msg.ProposalNum)
	// Optionally, update local state to reflect the chosen value
}

func main() {
	// Parse command-line flags
	hostfile := flag.String("h", "", "Path to hostsfile")
	value := flag.String("v", "", "Initial value to propose")
	delay := flag.Int("t", 0, "Delay before starting proposal (for proposer)")
	flag.Parse()

	if *hostfile == "" {
		os.Exit(1)
	}

	// Assign global variables
	initialValue = *value

	// Read hostsfile and set roles
	if err := readHostsFile(*hostfile); err != nil {
		os.Exit(1)
	}

	// Initialize round based on proposer ID
	if role.IsProposer {
		round = role.ProposerID * 10
	}

	// Start the listener in a separate goroutine so it runs concurrently
	go startListener()

	// Delay if required
	if *delay > 0 {
		time.Sleep(time.Duration(*delay) * time.Second)
	}

	// Start Paxos protocol if this peer is a proposer
	if role.IsProposer {
		proposerSendPrepare()
	}

	// Keep the main function alive to allow listener to continue running
	select {}
}
