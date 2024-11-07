
Paxos Algorithm Implementation Report

Introduction

This report outlines the implementation of the Paxos consensus algorithm in a distributed system using Go (Golang). The system is designed to handle multiple proposers and acceptors, ensuring that all nodes reach a consensus on a single value even in the presence of concurrent proposals. The implementation adheres to the Paxos protocol specifications and handles the nuances of message passing and state management in a distributed environment.

System Architecture

The system comprises three main roles:
	•	Proposers: Initiate proposals to suggest a value for consensus.
	•	Acceptors: Receive proposals and vote to accept or reject them based on the protocol.
	•	Learners: Learn the chosen value after consensus is reached.

Components and Interactions

	1.	Proposers:
	•	Generate unique proposal numbers.
	•	Send prepare and accept messages to acceptors.
	•	Handle prepare ack and accept ack responses.
	•	Decide when a value is chosen based on acknowledgments.
	2.	Acceptors:
	•	Maintain a single state with promisedProposal, acceptedProposal, and acceptedValue.
	•	Respond to prepare messages with prepare ack.
	•	Respond to accept messages with accept ack.
	•	Ensure the protocol’s safety by not accepting proposals with lower numbers than promised.
	3.	Learners:
	•	Receive chose messages indicating the value chosen.
	•	Update their state accordingly (implementation in this project focuses on proposers and acceptors).

Message Flow

	1.	Prepare Phase:
	•	Proposer sends prepare messages to a majority of acceptors.
	•	Acceptors respond with prepare ack, including any previously accepted proposal.
	2.	Accept Phase:
	•	Upon receiving a majority of prepare ack, the proposer sends accept messages.
	•	Acceptors respond with accept ack if the proposal number is acceptable.
	3.	Chosen Value:
	•	Once the proposer receives a majority of accept ack, it sends a chose message to all nodes.
	•	All nodes update their state with the chosen value.

State Diagrams

Proposer State Diagram

[Start] --> [Prepare Phase] --(majority prepare acks)--> [Accept Phase] --(majority accept acks)--> [Value Chosen]

	•	Prepare Phase: Proposer sends prepare messages.
	•	Accept Phase: Proposer sends accept messages after majority prepare ack.
	•	Value Chosen: Proposer sends chose message after majority accept ack.

Acceptor State Diagram

[Idle] --(receive prepare)--> [Promise] --(receive accept)--> [Accept] --(receive higher prepare)--> [Update Promise]

	•	Idle: Initial state, waiting for messages.
	•	Promise: Upon receiving a prepare with a higher proposal number, updates promisedProposal.
	•	Accept: Accepts the proposal if it meets criteria.
	•	Update Promise: If a higher proposal number is received.

Design Decisions

Unique Proposal Numbers

	•	Strategy: Proposal numbers are generated using a combination of a round number and the proposer’s unique ID:

proposalNum = (round * 10) + proposerID


	•	Reasoning: Ensures that proposal numbers are globally unique and monotonically increasing, which is critical for the correctness of the Paxos algorithm.

Single State in Acceptors

	•	Approach: Acceptors maintain a single state regardless of the number of proposers.
	•	Justification: Simplifies the acceptor logic and aligns with the Paxos protocol, where acceptors should not have separate states for different proposers.

Role-Based Message Routing

	•	Implementation: Proposers send messages only to acceptors with matching roles specified in the hostsfile.
	•	Benefit: Allows for flexible configuration and testing scenarios where proposers may interact with specific subsets of acceptors.

Message Logging and Formatting

	•	Format: All messages are logged in a JSON format specified in the project requirements.
	•	Consistency: Ensures that all nodes produce logs that are easy to parse and analyze for debugging and verification.

Implementation Issues and Resolutions

Handling Concurrent Proposals

	•	Issue: When multiple proposers initiate proposals concurrently, ensuring that the system reaches consensus on a single value.
	•	Resolution:
	•	By assigning higher initial proposal numbers to proposers with higher IDs, we reduce the likelihood of proposal conflicts.
	•	Implemented logic for proposers to adopt accepted values from prepare ack messages to maintain consistency.

Synchronization and Race Conditions

	•	Issue: Potential race conditions in updating shared variables like acknowledgment counts.
	•	Resolution:
	•	Utilized mutex locks (sync.Mutex) to protect critical sections where shared variables are updated.
	•	Ensured that state changes are thread-safe, especially in the context of Goroutines handling network connections.

Message Passing and Network Communication

	•	Issue: Reliable message passing over TCP connections with proper error handling.
	•	Resolution:
	•	Implemented robust error handling for network operations (dialing, encoding, decoding).
	•	Ensured that connections are closed properly to prevent resource leaks.

Timing and Delays

	•	Issue: Controlling the timing of proposal initiation to simulate specific test cases.
	•	Resolution:
	•	Introduced a -t command-line flag to specify delays for proposers.
	•	Used time.Sleep to delay the start of proposal processes.

Matching Acceptors Based on Roles

	•	Issue: Proposers needed to send messages only to acceptors with matching acceptor roles, as specified in the hostsfile.
	•	Resolution:
	•	Parsed acceptor roles from the hostsfile and stored them in the Role struct.
	•	Implemented checks in proposers to ensure messages are sent only to acceptors with matching roles.

Adherence to Message Format Requirements

	•	Issue: Ensuring that all messages conform to the specified JSON format without extra fields.
	•	Resolution:
	•	Standardized the Message struct to include only the required fields.
	•	Removed any unnecessary fields like proposer_role when not needed.

Conclusion

The implementation successfully demonstrates the Paxos algorithm’s ability to reach consensus in a distributed system with multiple proposers and acceptors. Through careful design decisions and attention to protocol specifics, the system handles concurrent proposals and ensures that all nodes agree on a single value. The use of Go’s concurrency features and standard library support for networking facilitates an efficient and robust implementation.
