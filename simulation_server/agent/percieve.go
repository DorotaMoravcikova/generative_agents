package agent

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

func (p *Persona) percieve(m *maze.Maze) []memory.NodeId {
	nearbyTiles := m.GetNearbyTiles(p.state.Position, p.state.VisionRadius)

	for _, pos := range nearbyTiles {
		tile := m.GetTile(pos)
		p.spatialMemory.Register(tile.Path)
	}

	// We will only consider events occuring in the same arena as the agent is currently in.
	currentArenaPath := m.GetTile(p.state.Position).Path.AtLevel(memory.PathLevelArena)

	// As objects might extend across multiple tiles, events could be registered multiple times.
	// We only want to percieve events once, so we deduplicate.
	perceptEventSet := make(map[maze.Event]struct{})
	// Events closer by should get priority over events further away so we order them.
	perceptEvents := []struct {
		event    maze.Event
		distance float64
	}{}
	for _, pos := range nearbyTiles {
		tile := m.GetTile(pos)
		if len(tile.Events) == 0 {
			continue
		}
		if !tile.Path.AtLevel(memory.PathLevelArena).Matches(currentArenaPath) {
			continue
		}

		distance := p.state.Position.EuclidianDistance(pos)

		for event := range tile.Events {
			if _, ok := perceptEventSet[event]; ok {
				continue
			}

			perceptEventSet[event] = struct{}{}
			perceptEvents = append(perceptEvents, struct {
				event    maze.Event
				distance float64
			}{event, distance})
		}
	}

	slices.SortFunc(perceptEvents, func(a, b struct {
		event    maze.Event
		distance float64
	},
	) int {
		return cmp.Compare(a.distance, b.distance)
	})

	percievedMax := min(p.state.AttentionBandwidth, len(perceptEvents))

	// We finally have all the events we might want to react to
	percievedEvents := make([]maze.Event, 0, percievedMax)
	for _, ev := range perceptEvents[:percievedMax] {
		percievedEvents = append(percievedEvents, ev.event)
	}

	memories := make([]memory.NodeId, 0, len(percievedEvents))
	for _, percievedEvent := range percievedEvents {
		if percievedEvent.SPO.Predicate == "" {
			percievedEvent.SPO.Predicate = "is"
			percievedEvent.SPO.Object = "idle"
			percievedEvent.Description = "idle"
		}
		percievedEvent.Description = fmt.Sprintf("%s is %s", percievedEvent.SPO.Subject, percievedEvent.Description)

		// Skip events we have recently percieved already
		if _, ok := p.associativeMemory.GetLatestEventSPOs(p.state.Retention)[percievedEvent.SPO]; ok {
			continue
		}

		keywords := make([]string, 0, 2)

		subject := memory.ParsePath(percievedEvent.SPO.Subject).Base()
		object := memory.ParsePath(percievedEvent.SPO.Object).Base()
		keywords = append(keywords, subject)
		keywords = append(keywords, object)

		var embedding []float64 = p.GetEmbedding(percievedEvent.Description)

		importance := p.cognition.GenerateImportanceScore(p, memory.NodeTypeEvent, percievedEvent.Description)
		valence := p.cognition.GenerateValenceScore(p, memory.NodeTypeEvent, percievedEvent.Description)

		chatNodes := make([]memory.NodeId, 0, 1)
		if subject == p.name && percievedEvent.SPO.Predicate == "chat with" {
			currentEvent := p.state.ActivitySPO

			var chatEmbedding []float64 = p.GetEmbedding(p.state.ActivityDescription)

			chatImportance := p.cognition.GenerateImportanceScoreChat(p, p.state.Chat, p.state.ActivityDescription)
			chatValence := p.cognition.GenerateValenceScoreChat(p, p.state.Chat, p.state.ActivityDescription)

			chatNode := p.addChatToMemory(currentEvent, p.state.ActivityDescription, keywords, chatImportance, chatValence, p.state.Chat, p.state.CurrentTime, nil, p.state.ActivityDescription, chatEmbedding)
			chatNodes = append(chatNodes, chatNode.Id)
		}

		memories = append(memories, p.addEventToMemory(percievedEvent, keywords, importance, valence, chatNodes, percievedEvent.Description, embedding).Id)

	}

	return memories
}
