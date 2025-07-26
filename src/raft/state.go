package raft

func (rf *Raft) updateStateForElection() int {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.currentTerm++
	rf.role = Follower
	rf.votedFor = rf.me
	return rf.currentTerm
}

// returns true if the election result is accepted
func (rf *Raft) updateForElectionResult(electionResult ElectionResult) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if electionResult.Term == rf.currentTerm && electionResult.VoteGranted {
		rf.role = Leader
		rf.votedFor = rf.me
		return true
	}
	if electionResult.Term > rf.currentTerm {
		rf.currentTerm = electionResult.Term
		rf.role = Follower
		rf.votedFor = electionResult.VotedFor
	}
	return false
}

// example RequestVote RPC handler.
func (rf *Raft) updateOnhandleRequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// defer func() {
	// 	// fmt.Printf("server %d: RequestVote, args: %+v, reply: %+v\n", rf.me, args, reply)
	// }()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.role = Follower
		rf.votedFor = args.CandidateId
		rf.sendToDegradeChan()

		reply.Term = rf.currentTerm
		reply.VoteGranted = true
		return
	}

	if args.Term == rf.currentTerm && rf.votedFor == args.CandidateId {
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
		return
	}

	// has voated for a different candidate
	reply.Term = rf.currentTerm
	reply.VoteGranted = false
}

func (rf *Raft) updateForAppendEntries(reply *AppendEntriesResult) bool {
	// todo: simplify for 2A only
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.role = Follower
		rf.votedFor = reply.NodeIdWithHigherTerm
		return true
	}
	return false
}
