package dht

import "sync"

type peer struct {
	node Node
	//<lastPublished>,
	//<originallyPublished>
	//	<originalPublisherID>
}

type peerStore struct {
	// map of blob hashes to (map of node IDs to bools)
	hashes map[Bitmap]map[Bitmap]bool
	// map of node IDs to peers
	nodeInfo map[Bitmap]peer
	lock     sync.RWMutex
}

func newPeerStore() *peerStore {
	return &peerStore{
		hashes:   make(map[Bitmap]map[Bitmap]bool),
		nodeInfo: make(map[Bitmap]peer),
	}
}

func (s *peerStore) Upsert(blobHash Bitmap, node Node) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.hashes[blobHash]; !ok {
		s.hashes[blobHash] = make(map[Bitmap]bool)
	}
	s.hashes[blobHash][node.id] = true
	s.nodeInfo[node.id] = peer{node: node}
}

func (s *peerStore) Get(blobHash Bitmap) []Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var nodes []Node
	if ids, ok := s.hashes[blobHash]; ok {
		for id := range ids {
			peer, ok := s.nodeInfo[id]
			if !ok {
				panic("node id in IDs list, but not in nodeInfo")
			}
			nodes = append(nodes, peer.node)
		}
	}
	return nodes
}

func (s *peerStore) CountStoredHashes() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.hashes)
}
