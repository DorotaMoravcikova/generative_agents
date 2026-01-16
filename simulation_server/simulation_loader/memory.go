package simulationloader

import (
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

func memoryNodeIterator[T any](m map[string]T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		n := len(m)
		for i := 1; i <= n; i += 1 {
			key := "node_" + strconv.Itoa(i)
			if v, ok := m[key]; ok {
				if !yield(i, v) {
					return
				}
			}
		}
	}
}

func LoadSpatialMemory(memFile string) (*memory.Spatial, error) {
	content, err := os.ReadFile(memFile)
	if err != nil {
		return nil, err
	}

	locs := map[string]map[string]map[string][]string{}
	if err := json.Unmarshal(content, &locs); err != nil {
		return nil, err
	}

	mem := memory.NewSpatial()
	for w, sectors := range locs {
		world := memory.NewPath(memory.PathWithWorld(w))
		mem.Register(world)

		for s, arenas := range sectors {
			sector := world.Copy(memory.PathWithSector(s))
			mem.Register(sector)

			for a, objects := range arenas {
				arena := sector.Copy(memory.PathWithArena(a))
				mem.Register(arena)

				for _, o := range objects {
					object := arena.Copy(memory.PathWithObject(o))
					mem.Register(object)
				}
			}
		}
	}

	return mem, nil
}

func extractEvidence(filling interface{}) ([]memory.NodeId, error) {
	evidence := []memory.NodeId{}
	switch c := filling.(type) {
	case string:
		var id int
		if _, err := fmt.Sscanf(c, "node_%d", &id); err != nil {
			return nil, fmt.Errorf("could not parse node id: %w", err)
		}
		evidence = append(evidence, memory.NodeId(id))
	case []string:
		for _, c := range c {
			var id int
			if _, err := fmt.Sscanf(c, "node_%d", &id); err != nil {
				return nil, fmt.Errorf("could not parse node id: %w", err)
			}
			evidence = append(evidence, memory.NodeId(id))
		}
	case []interface{}:
		for _, c := range c {
			idStr, ok := c.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected element type in filling list: %T", c)
			}
			var id int
			if _, err := fmt.Sscanf(idStr, "node_%d", &id); err != nil {
				return nil, fmt.Errorf("could not parse node id: %w", err)
			}
			evidence = append(evidence, memory.NodeId(id))
		}
	default:
		if filling != nil {
			return nil, fmt.Errorf("unexpected filling type in memory store: %T", filling)
		}
	}

	return evidence, nil
}

func extractChat(filling interface{}) ([]memory.Utterance, error) {
	chat := make([]memory.Utterance, 0)

	switch filling := filling.(type) {
	case [][]string:
		for _, utt := range filling {
			chat = append(chat, memory.Utterance{
				Speaker:  utt[0],
				Sentence: utt[1],
			})
		}
	case []interface{}:
		for _, f := range filling {
			switch f := f.(type) {
			case []string:
				chat = append(chat, memory.Utterance{
					Speaker:  f[0],
					Sentence: f[1],
				})
			case []interface{}:
				speaker, ok := f[0].(string)
				if !ok {
					return nil, fmt.Errorf("unexpected chat memory filling element type: %T", f[0])
				}
				sentence, ok := f[1].(string)
				if !ok {
					return nil, fmt.Errorf("unexpected chat memory filling element type: %T", f[0])
				}
				chat = append(chat, memory.Utterance{
					Speaker:  speaker,
					Sentence: sentence,
				})
			default:
				return nil, fmt.Errorf("unexpected chat memory filling element type: %T", f)
			}
		}
	default:
		return nil, fmt.Errorf("unexpected chat memory filling type: %T", filling)
	}

	return chat, nil
}

func LoadAssociativeMemory(folder string) (*memory.Associative, error) {
	content, err := os.ReadFile(path.Join(folder, "embeddings.json"))
	if err != nil {
		return nil, fmt.Errorf("could not read embeddings file: %w", err)
	}

	embeddings := map[string][]float64{}
	if err = json.Unmarshal(content, &embeddings); err != nil {
		return nil, fmt.Errorf("could not unmarshal embeddings json: %w", err)
	}

	content, err = os.ReadFile(path.Join(folder, "kw_strength.json"))
	if err != nil {
		return nil, fmt.Errorf("could not read keyword strength file: %w", err)
	}

	kws := KwStength{
		Thoughts: map[string]int{},
		Events:   map[string]int{},
	}
	if err = json.Unmarshal(content, &kws); err != nil {
		return nil, fmt.Errorf("could not unmarshal keyword strength json: %w", err)
	}

	content, err = os.ReadFile(path.Join(folder, "nodes.json"))
	if err != nil {
		return nil, fmt.Errorf("could not read memory nodes file: %w", err)
	}

	memories := map[string]MemoryNode{}
	if err = json.Unmarshal(content, &memories); err != nil {
		return nil, fmt.Errorf("could not unmarshal memory nodes json: %w", err)
	}

	store := memory.NewAssociative(embeddings, kws.Events, kws.Thoughts)
	for _, mem := range memoryNodeIterator(memories) {
		switch mem.Type {
		case "event":
			evidence, err := extractEvidence(mem.Filling)
			if err != nil {
				return nil, err
			}

			var expiration *time.Time = nil
			if mem.Expiration != nil {
				exp := time.Time(*mem.Expiration)
				expiration = &exp
			}

			store.AddEvent(
				memory.SPO{
					Subject:   mem.Subject,
					Predicate: mem.Predicate,
					Object:    mem.Object,
				},
				mem.Description,
				mem.Keywords,
				mem.Poignancy,
				mem.Valence,
				evidence,
				time.Time(mem.Created),
				expiration,
				mem.EmbeddingKey,
				embeddings[mem.EmbeddingKey],
			)
		case "chat":
			chat, err := extractChat(mem.Filling)
			if err != nil {
				return nil, err
			}

			var expiration *time.Time = nil
			if mem.Expiration != nil {
				exp := time.Time(*mem.Expiration)
				expiration = &exp
			}

			store.AddChat(
				memory.SPO{
					Subject:   mem.Subject,
					Predicate: mem.Predicate,
					Object:    mem.Object,
				},
				mem.Description,
				mem.Keywords,
				mem.Poignancy,
				mem.Valence,
				chat,
				time.Time(mem.Created),
				expiration,
				mem.EmbeddingKey,
				embeddings[mem.EmbeddingKey],
			)
		case "thought":
			evidence, err := extractEvidence(mem.Filling)
			if err != nil {
				return nil, err
			}

			var expiration *time.Time = nil
			if mem.Expiration != nil {
				exp := time.Time(*mem.Expiration)
				expiration = &exp
			}

			store.AddThought(
				memory.SPO{
					Subject:   mem.Subject,
					Predicate: mem.Predicate,
					Object:    mem.Object,
				},
				mem.Description,
				mem.Keywords,
				mem.Poignancy,
				mem.Valence,
				evidence,
				time.Time(mem.Created),
				expiration,
				mem.EmbeddingKey,
				embeddings[mem.EmbeddingKey],
			)
		default:
			panic(fmt.Sprintf("unknown memory type: %s", mem.Type))
		}
	}

	return store, nil
}
