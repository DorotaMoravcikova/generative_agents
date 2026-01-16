package memory

import (
	"fmt"
	"strings"
	"time"
)

type SPO struct {
	Subject   string
	Predicate string
	Object    string
}

type NodeType int

const (
	NodeTypeInvalid NodeType = iota
	NodeTypeThought
	NodeTypeEvent
	NodeTypeChat
)

func (t NodeType) ToString() string {
	switch t {
	case NodeTypeChat:
		return "chat"
	case NodeTypeEvent:
		return "event"
	case NodeTypeThought:
		return "thought"
	default:
		panic(fmt.Sprintf("unexpected memory.NodeType: %#v", t))
	}
}

type Utterance struct {
	// The name of the persona who says this utterance
	Speaker string
	// What the persona actually says
	Sentence string
}

type NodeId int

type ConceptNode struct {
	Id NodeId

	// It's basically the same as Id but I'm keeping it now for posterity
	NodeCount int
	// This node is the n'th node of its type
	TypeCount int

	Type NodeType
	// The "depth" of this thought, on how many layers of other thoughts it depends
	Depth int

	Created      time.Time
	LastAccessed time.Time
	// Expiration does not have to be set
	Expiration *time.Time

	// The core parts of the concept node there might be a better way to encode this,
	// at least the subject.
	Subject   string
	Predicate string
	Object    string

	Description  string
	EmbeddingKey string
	Importance   int
	Valence      int

	// Set of keywords related to this memory
	Keywords []string

	// All nodes this node was used to construct
	Evidence []NodeId
	// If this is a chat node what was said in the conversation
	Chat []Utterance
}

func (node ConceptNode) SPOSummary() SPO {
	return SPO{node.Subject, node.Predicate, node.Object}
}

type Associative struct {
	// All nodes in the memory stream, indexed by NodeId
	nodes []ConceptNode

	// Nodes sorted by type they are stored reverse order,
	// newer nodes will be stored at lower indices
	events   []NodeId
	thoughts []NodeId
	chats    []NodeId

	kwToEvents   map[string][]NodeId
	kwToThoughts map[string][]NodeId
	kwToChats    map[string][]NodeId

	kwStrengthEvents   map[string]int
	kwStrengthThoughts map[string]int

	embeddings map[string][]float64
}

func NewAssociative(embeddings map[string][]float64, kwStrengthEvents map[string]int, kwStrengthThoughts map[string]int) *Associative {
	// Size to initialize memory store slices to
	initialMemorySize := 5

	return &Associative{
		// The node ID's start at 1, so create an empty node at index 0
		nodes:              make([]ConceptNode, 1, initialMemorySize),
		events:             make([]NodeId, 0, initialMemorySize),
		thoughts:           make([]NodeId, 0, initialMemorySize),
		chats:              make([]NodeId, 0, initialMemorySize),
		kwToEvents:         make(map[string][]NodeId),
		kwToThoughts:       make(map[string][]NodeId),
		kwToChats:          make(map[string][]NodeId),
		kwStrengthEvents:   kwStrengthEvents,
		kwStrengthThoughts: kwStrengthThoughts,
		embeddings:         embeddings,
	}
}

func (store *Associative) Embeddings() map[string][]float64 {
	return store.embeddings
}

func (store *Associative) EventKeywordStrength() map[string]int {
	return store.kwStrengthEvents
}

func (store *Associative) ThoughtKeywordStrength() map[string]int {
	return store.kwStrengthThoughts
}

func (store *Associative) Nodes() []ConceptNode {
	return store.nodes[1:]
}

func (store *Associative) GetNode(node NodeId) ConceptNode {
	return store.nodes[node]
}

func (store *Associative) UpdateNode(node NodeId, update func(*ConceptNode)) {
	update(&store.nodes[node])
}

func (store *Associative) GetEmbedding(str string) ([]float64, bool) {
	e, ok := store.embeddings[str]
	return e, ok
}

func (store *Associative) GetEmbeddingByNodeId(id NodeId) ([]float64, bool) {
	e, ok := store.embeddings[store.GetNode(id).EmbeddingKey]
	return e, ok
}

func (store *Associative) SaveEmbedding(str string, embedding []float64) {
	store.embeddings[str] = embedding
}

func (store *Associative) AddEvent(spo SPO, description string, keywords []string, importance, valence int, evidence []NodeId, created time.Time, expiration *time.Time, embeddingKey string, embedding []float64) ConceptNode {
	nodeCount := len(store.nodes)
	typeCount := len(store.events)
	nodeType := NodeTypeEvent
	nodeId := NodeId(nodeCount)

	depth := 0

	// In the paper code they do some cleanup here,
	// I'm not sure why they do it so I won't add it for now

	node := ConceptNode{
		Id:           nodeId,
		NodeCount:    nodeCount,
		TypeCount:    typeCount,
		Type:         nodeType,
		Depth:        depth,
		Created:      created,
		LastAccessed: created,
		Expiration:   expiration,
		Subject:      spo.Subject,
		Predicate:    spo.Predicate,
		Object:       spo.Object,
		Description:  description,
		EmbeddingKey: embeddingKey,
		Importance:   importance,
		Valence:      valence,
		Keywords:     keywords,
		Evidence:     evidence,
		Chat:         make([]Utterance, 0),
	}

	store.nodes = append(store.nodes, node)
	store.events = append([]NodeId{node.Id}, store.events...)

	for i := range keywords {
		keywords[i] = strings.ToLower(keywords[i])
	}
	for _, kw := range keywords {
		if kws, ok := store.kwToEvents[kw]; ok {
			store.kwToEvents[kw] = append([]NodeId{node.Id}, kws...)
		} else {
			store.kwToEvents[kw] = []NodeId{node.Id}
		}
	}

	if spo.Predicate != "is" && spo.Object != "idle" {
		for _, kw := range keywords {
			// Zero value is well 0
			store.kwStrengthEvents[kw] += 1
		}
	}

	store.embeddings[embeddingKey] = embedding

	return node
}

func (store *Associative) AddThought(spo SPO, description string, keywords []string, importance, valence int, evidence []NodeId, created time.Time, expiration *time.Time, embeddingKey string, embedding []float64) ConceptNode {
	nodeCount := len(store.nodes)
	typeCount := len(store.thoughts)
	nodeType := NodeTypeThought
	nodeId := NodeId(nodeCount)

	depth := 1
	maxDepth := 0
	for _, nodeId := range evidence {
		if store.nodes[nodeId].Depth > maxDepth {
			maxDepth = store.nodes[nodeId].Depth
		}
	}
	depth += maxDepth

	node := ConceptNode{
		Id:           nodeId,
		NodeCount:    nodeCount,
		TypeCount:    typeCount,
		Type:         nodeType,
		Depth:        depth,
		Created:      created,
		LastAccessed: created,
		Expiration:   expiration,
		Subject:      spo.Subject,
		Predicate:    spo.Predicate,
		Object:       spo.Object,
		Description:  description,
		EmbeddingKey: embeddingKey,
		Importance:   importance,
		Valence:      valence,
		Keywords:     keywords,
		Evidence:     evidence,
		Chat:         make([]Utterance, 0),
	}

	store.nodes = append(store.nodes, node)
	store.thoughts = append([]NodeId{node.Id}, store.thoughts...)

	for i := range keywords {
		keywords[i] = strings.ToLower(keywords[i])
	}
	for _, kw := range keywords {
		if kws, ok := store.kwToThoughts[kw]; ok {
			store.kwToThoughts[kw] = append([]NodeId{node.Id}, kws...)
		} else {
			store.kwToThoughts[kw] = []NodeId{node.Id}
		}
	}

	if spo.Predicate != "is" && spo.Object != "idle" {
		for _, kw := range keywords {
			// Zero value is well 0
			store.kwStrengthThoughts[kw] += 1
		}
	}

	store.embeddings[embeddingKey] = embedding

	return node
}

func (store *Associative) AddChat(spo SPO, description string, keywords []string, importance, valence int, chat []Utterance, created time.Time, expiration *time.Time, embeddingKey string, embedding []float64) ConceptNode {
	nodeCount := len(store.nodes)
	typeCount := len(store.thoughts)
	nodeType := NodeTypeChat
	nodeId := NodeId(nodeCount)

	depth := 0

	node := ConceptNode{
		Id:           nodeId,
		NodeCount:    nodeCount,
		TypeCount:    typeCount,
		Type:         nodeType,
		Depth:        depth,
		Created:      created,
		LastAccessed: created,
		Expiration:   expiration,
		Subject:      spo.Subject,
		Predicate:    spo.Predicate,
		Object:       spo.Object,
		Description:  description,
		EmbeddingKey: embeddingKey,
		Importance:   importance,
		Valence:      valence,
		Keywords:     keywords,
		Evidence:     make([]NodeId, 0),
		Chat:         chat,
	}

	store.nodes = append(store.nodes, node)
	store.chats = append([]NodeId{node.Id}, store.chats...)

	for i := range keywords {
		keywords[i] = strings.ToLower(keywords[i])
	}
	for _, kw := range keywords {
		if kws, ok := store.kwToChats[kw]; ok {
			store.kwToChats[kw] = append([]NodeId{node.Id}, kws...)
		} else {
			store.kwToChats[kw] = []NodeId{node.Id}
		}
	}

	store.embeddings[embeddingKey] = embedding

	return node
}

func (store *Associative) GetLatestEventSPOs(n int) map[SPO]struct{} {
	events := make(map[SPO]struct{})

	if len(store.events) < n {
		n = len(store.events)
	}
	for _, nodeId := range store.events[:n] {
		events[store.nodes[nodeId].SPOSummary()] = struct{}{}
	}

	return events
}

func (store *Associative) RetrieveRelevantEvents(subject string, predicate string, object string) map[NodeId]struct{} {
	ret := map[NodeId]struct{}{}

	for _, i := range [3]string{subject, predicate, object} {
		if thoughts, ok := store.kwToThoughts[i]; ok {
			for _, thought := range thoughts {
				ret[thought] = struct{}{}
			}
		}
	}

	return ret
}

func (store *Associative) RetrieveRelevantThoughts(subject string, predicate string, object string) map[NodeId]struct{} {
	ret := map[NodeId]struct{}{}

	for _, i := range [3]string{subject, predicate, object} {
		if thoughts, ok := store.kwToThoughts[i]; ok {
			for _, thought := range thoughts {
				ret[thought] = struct{}{}
			}
		}
	}

	return ret
}

func (store *Associative) GetLatestEventIds() []NodeId {
	return store.events
}

func (store *Associative) GetLatestThoughtIds() []NodeId {
	return store.thoughts
}

func (store *Associative) GetLastChat(name string) (NodeId, bool) {
	if chats, ok := store.kwToChats[name]; ok {
		return chats[0], true
	}

	return 0, false
}
