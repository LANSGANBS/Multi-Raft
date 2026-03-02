package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
	peerID := 0
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &peerID)
	}
	
	peers := []string{
		"127.0.0.1:50051",
		"127.0.0.1:50052",
		"127.0.0.1:50053",
	}
	
	dataDir := fmt.Sprintf("/tmp/raft-node-%d", peerID)
	os.RemoveAll(dataDir)
	
	storage, err := raft.NewBadgerRaftStorage(dataDir)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()
	
	transport := raft.NewGrpcTransport(&raft.TransportConfig{
		Peers:      peers,
		PeerID:     peerID,
		ListenAddr: peers[peerID],
	})
	
	applyCh := make(chan raft.ApplyMsg, 100)
	
	config := raft.DefaultConfig()
	node := raft.NewNode(config, transport, storage, applyCh)
	
	if err := transport.Start(); err != nil {
		log.Fatalf("Failed to start transport: %v", err)
	}
	
	go func() {
		for msg := range applyCh {
			if msg.CommandValid {
				fmt.Printf("[Node %d] Applied command at index %d: %v\n", peerID, msg.CommandIndex, msg.Command)
			} else if msg.SnapshotValid {
				fmt.Printf("[Node %d] Applied snapshot at index %d\n", peerID, msg.SnapshotIndex)
			}
		}
	}()
	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-sigCh:
			fmt.Printf("\n[Node %d] Shutting down...\n", peerID)
			node.Stop()
			return
			
		case <-ticker.C:
			term, isLeader := node.GetState()
			if isLeader {
				cmd := fmt.Sprintf("command-from-node-%d", peerID)
				index, term, ok := node.Start(cmd)
				if ok {
					fmt.Printf("[Node %d] Leader proposed command at index %d, term %d\n", peerID, index, term)
				}
			} else {
				fmt.Printf("[Node %d] Follower in term %d\n", peerID, term)
			}
		}
	}
}
