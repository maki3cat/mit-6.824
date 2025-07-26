package raft

import (
	"math/rand"
	"sync"
	"time"
)


type AppendEntriesResult struct {
	Term                 int
	Success              bool
	NodeIdWithHigherTerm int
}

// simplified version since we only need leader election in 2A
func (rf *Raft) sendHeartbeat(currentTerm int, resultChan chan *AppendEntriesResult) {
	faninCh := make(chan *AppendEntriesResult, len(rf.peers)-1)
	wg := sync.WaitGroup{}
	callOne := func(peerId int) {
		defer wg.Done()
		reply := AppendEntriesReply{}
		args := AppendEntriesArgs{
			Term:     currentTerm,
			LeaderId: rf.me,
		}
		called := rf.sendAppendEntries(peerId, &args, &reply)
		if !called {
			// todo: should be better
			faninCh <- &AppendEntriesResult{
				Term:    currentTerm,
				Success: false,
			}
			return
		}
		if reply.Term > currentTerm {
			faninCh <- &AppendEntriesResult{
				Term:                 reply.Term,
				Success:              false,
				NodeIdWithHigherTerm: peerId,
			}
			return
		}
		faninCh <- &AppendEntriesResult{
			Term:                 currentTerm,
			Success:              true,
			NodeIdWithHigherTerm: -1,
		}
	}
	for peerId := 0; peerId < len(rf.peers); peerId += 1 {
		if peerId == rf.me {
			continue
		}
		wg.Add(1)
		go callOne(peerId)
	}
	go func() {
		wg.Wait()
		close(faninCh)
	}()
	// todo: to simplify the logic here
	for singleResult := range faninCh {
		if singleResult.Term > currentTerm {
			resultChan <- &AppendEntriesResult{
				Term:                 singleResult.Term,
				Success:              false,
				NodeIdWithHigherTerm: singleResult.NodeIdWithHigherTerm,
			}
			return
		}
	}
	resultChan <- &AppendEntriesResult{
		Term:                 currentTerm,
		Success:              true,
		NodeIdWithHigherTerm: rf.me,
	}
}

type ElectionResult struct {
	OnError     bool
	Term        int
	VoteGranted bool
	VotedFor    int
}

func (rf *Raft) startOneRoundElection(currentTerm int, electionResultCh chan ElectionResult) {
	faninCh := make(chan ElectionResult, len(rf.peers)-1)
	wg := sync.WaitGroup{}
	callOne := func(peerId int) {
		defer wg.Done()
		reply := RequestVoteReply{}
		args := RequestVoteArgs{
			Term:        currentTerm,
			CandidateId: rf.me,
			// TODO: get last log index and term
			LastLogIndex: 0,
			LastLogTerm:  0,
		}
		called := rf.sendRequestVote(peerId, &args, &reply)
		if !called {
			faninCh <- ElectionResult{
				OnError: true,
			}
		} else {
			res := ElectionResult{
				OnError:  false,
				VotedFor: rf.me,
			}
			if reply.Term > currentTerm {
				res.Term = reply.Term
				res.VoteGranted = false
				res.VotedFor = peerId
			} else if reply.VoteGranted {
				res.VoteGranted = true
				res.VotedFor = peerId
				res.Term = currentTerm
			}
			faninCh <- res
		}
	}
	for peerId := 0; peerId < len(rf.peers); peerId += 1 {
		if peerId == rf.me {
			continue
		}
		wg.Add(1)
		go callOne(peerId)
	}
	go func() {
		wg.Wait()
		close(faninCh)
	}()
	numVotes := 1 // self vote
	for singleResult := range faninCh {
		// error
		if singleResult.OnError {
			continue
		}
		// has vote
		if singleResult.VoteGranted {
			numVotes++
			if numVotes > len(rf.peers)/2 {
				electionResultCh <- ElectionResult{
					Term:        currentTerm,
					VoteGranted: true,
					VotedFor:    rf.me,
				}
				return
			}
			continue
		}
		// no vote and others has a higher term
		if singleResult.Term > currentTerm {
			electionResultCh <- ElectionResult{
				Term:        singleResult.Term,
				VoteGranted: false,
				VotedFor:    singleResult.VotedFor,
			}
			return
		}
	}
	electionResultCh <- ElectionResult{
		Term:        currentTerm,
		VoteGranted: false,
		VotedFor:    rf.me,
	}
}

func getElectionTimeout() time.Duration {
	return time.Duration(300+rand.Intn(200)) * time.Millisecond
}
func getHeartbeatTimeout() time.Duration {
	return 50 * time.Millisecond
}
