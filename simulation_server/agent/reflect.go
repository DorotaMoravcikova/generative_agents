package agent

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

func (p *Persona) generateFocalPoints() []string {
	nFocalPoints := 3

	nodes := make([]memory.NodeId, 0, len(p.associativeMemory.GetLatestEventIds())+len(p.associativeMemory.GetLatestThoughtIds()))
	nodes = append(nodes, p.associativeMemory.GetLatestEventIds()...)
	nodes = append(nodes, p.associativeMemory.GetLatestThoughtIds()...)

	nodes = slices.DeleteFunc(nodes, func(n memory.NodeId) bool {
		return strings.Contains(
			p.associativeMemory.GetNode(n).EmbeddingKey,
			"idle")
	})

	slices.SortFunc(nodes, func(a, b memory.NodeId) int {
		memA := p.associativeMemory.GetNode(a)
		memB := p.associativeMemory.GetNode(b)
		return memA.LastAccessed.Compare(memB.LastAccessed)
	})

	// NOTE(Friso): In the paper they say they reflect using the last 100 memories, in the code they use all memories made since the last reflection
	// This is a middle of the road solution
	n := max(p.state.ReflectionElements, 100)
	if n < 0 {
		n = 0
	} else if n >= len(nodes) {
		n = len(nodes)
	}

	return p.cognition.GenerateFocalPoints(p, nodes[len(nodes)-n:], nFocalPoints)
}

func (p *Persona) runReflect() {
	focalPoints := p.generateFocalPoints()
	retrieved := p.retrieveForFocalPoints(focalPoints)

	for _, nodes := range retrieved {
		thoughts := p.cognition.GenerateInsightAndEvidence(p, nodes, 5)
		for thought, evidence := range thoughts {
			created := p.state.CurrentTime
			expiration := p.state.CurrentTime.Add(time.Hour * 24 * 30)
			spo := p.cognition.GenerateActivitySPO(p, thought)
			keywords := []string{spo.Subject, spo.Predicate, spo.Object}
			importance := p.cognition.GenerateImportanceScore(p, memory.NodeTypeThought, thought)
			valence := p.cognition.GenerateValenceScore(p, memory.NodeTypeThought, thought)
			embedding := p.GetEmbedding(thought)

			p.addThoughtToMemory(spo, thought, keywords, importance, valence, evidence, created, &expiration, thought, embedding)
		}
	}
}

func (p *Persona) shouldReflect() bool {
	return p.state.CurrentReflectionTrigger < 1 &&
		len(p.associativeMemory.GetLatestEventIds())+len(p.associativeMemory.GetLatestThoughtIds()) != 0
}

func (p *Persona) resetReflectionTrigger() {
	p.state.CurrentReflectionTrigger = p.state.ReflectionTrigger
	p.state.ReflectionElements = 0
}

func (p *Persona) reflect() {
	if p.shouldReflect() {
		p.runReflect()
		p.resetReflectionTrigger()
	}

	if !p.state.ChatEndTime.IsZero() &&
		!p.state.CurrentTime.
			Add(10*time.Second).
			Before(p.state.ChatEndTime) {
		var evidence []memory.NodeId
		if id, ok := p.associativeMemory.GetLastChat(p.state.ChattingWith); ok {
			evidence = []memory.NodeId{id}
		}

		planningThought := p.cognition.GeneratePlanningThoughtAfterConversation(p, p.state.Chat)
		planningThought = fmt.Sprintf("For %s's planning: %s", p.name, planningThought)

		created := p.state.CurrentTime
		expired := p.state.CurrentTime.Add(time.Hour * 24 * 30)

		spo := p.cognition.GenerateActivitySPO(p, planningThought)
		keywords := []string{spo.Subject, spo.Predicate, spo.Object}

		importance := p.cognition.GenerateImportanceScore(p, memory.NodeTypeThought, planningThought)
		valence := p.cognition.GenerateValenceScore(p, memory.NodeTypeThought, planningThought)
		embedding := p.GetEmbedding(planningThought)

		p.addThoughtToMemory(spo, planningThought, keywords, importance, valence, evidence, created, &expired, planningThought, embedding)

		memoThought := p.cognition.GenerateMemoAfterConversation(p, p.state.Chat)
		memoThought = fmt.Sprintf("%s %s", p.name, memoThought)

		created2 := p.state.CurrentTime
		expired2 := p.state.CurrentTime.Add(time.Hour * 24 * 30)

		spo2 := p.cognition.GenerateActivitySPO(p, memoThought)
		keywords2 := []string{spo2.Subject, spo2.Predicate, spo2.Object}

		importance2 := p.cognition.GenerateImportanceScore(p, memory.NodeTypeThought, memoThought)
		valence2 := p.cognition.GenerateValenceScore(p, memory.NodeTypeThought, memoThought)
		embedding2 := p.GetEmbedding(memoThought)

		p.addThoughtToMemory(spo2, memoThought, keywords2, importance2, valence2, evidence, created2, &expired2, memoThought, embedding2)
	}
}
