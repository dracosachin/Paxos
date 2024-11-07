## Author

**Sachin Channabasavarajendra**  
**NUID: 002780819**


Build the Docker image:

```bash
docker build -t prj4 .
```

Test Case 1: Single Proposer

Objective

	•	Peer1 starts with value X.
	•	All acceptors should agree on value X.
	•	Verify basic consensus in the system.

Running Test Case 1

	1.	Start the Docker Containers:

```bash
docker-compose -f docker-compose-testcase1.yml up
```


	2.	Observe the Output:
	•	Each peer will print messages in the specified JSON format.
	•	Look for "action": "chose" messages to confirm the chosen value.

Expected Outcome

	•	Peer1 sends prepare and accept messages with value X.
	•	Peers 2, 3, 4 (acceptors) respond accordingly.
	•	Peer5 (learner) receives the chose message.
	•	All peers agree on value X.

Test Case 2: Two Proposers with Concurrent Proposals

Objective

	•	Peer1 starts with value X.
	•	Peer5 (proposer with the highest ID) starts with value Y after a delay.
	•	Peer5 should initiate its proposal after Peer3 sends the accept to Peer1’s proposal.
	•	All peers should agree on value X, demonstrating conflict resolution in Paxos.


Running Test Case 2

	1.	Start the Docker Containers:

```bash
docker-compose -f docker-compose-testcase2.yml up
```


	2.	Observe the Output:
	•	Monitor the logs of each peer.
	•	Peer5 starts its proposal after a delay (controlled by -t 10).
	•	Look for "action": "chose" messages to confirm the chosen value.

Expected Outcome

	•	Peer1 begins the proposal process with value X.
	•	Peers 2, 3, 4 (acceptors) respond to Peer1.
	•	After Peer3 sends an accept to Peer1, Peer5 starts proposing value Y.
	•	Acceptors may have already accepted Peer1’s proposal and may reject Peer5’s proposal due to lower proposal numbers.
	•	All peers agree on value X.

