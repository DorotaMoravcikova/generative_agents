package agent

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

func (p *Persona) retrieveForPerceptions(percieved []memory.NodeId) map[string]relevantNodes {
	retrieved := make(map[string]relevantNodes)
	for _, id := range percieved {
		event := p.associativeMemory.GetNode(id)

		thoughts := p.associativeMemory.RetrieveRelevantThoughts(event.Subject, event.Predicate, event.Object)
		events := p.associativeMemory.RetrieveRelevantEvents(event.Subject, event.Predicate, event.Object)

		retrieved[event.Description] = relevantNodes{currEvent: id, thoughts: thoughts, events: events}
	}

	return retrieved
}

func cosineSimilarity[V float32 | float64](a, b []V) float64 {
	if len(a) != len(b) {
		panic(fmt.Errorf("trying to compute the cosine similarity between vectors of different length: %d, %d", len(a), len(b)))
	}
	if len(a) == 0 {
		panic(fmt.Errorf("trying to compute the cosine similarity of empty vectors"))
	}

	var dot, na, nb float64
	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		na += ai * ai
		nb += bi * bi
	}

	if na == 0 || nb == 0 {
		panic(fmt.Errorf("zero norm vector in cosine similarity"))
	}

	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func normalizeMap[K comparable, V float32 | float64](m map[K]V, targetMin, targetMax V) map[K]V {
	var min, max V = V(math.NaN()), V(math.NaN())

	for _, v := range m {
		if math.IsNaN(float64(v)) {
			continue
		}

		if math.IsNaN(float64(min)) {
			min = v
		}
		if math.IsNaN(float64(max)) {
			max = v
		}

		if v < min {
			min = v
		}
		if max < v {
			max = v
		}
	}

	if math.IsNaN(float64(min)) || math.IsNaN(float64(max)) {
		panic(fmt.Errorf("map has NaN min (%f) or max (%f) value", min, max))
	}

	if max == min {
		for key := range m {
			m[key] = (targetMax - targetMin) / 2
		}
	} else {
		dist := max - min
		for key, val := range m {
			m[key] = ((val-min)*(targetMax-targetMin))/dist + targetMin
		}
	}

	return m
}

func highestNValues[K comparable, V float32 | float64](m map[K]V, n int) map[K]V {
	type KV struct {
		k K
		v V
	}

	values := make([]KV, 0, len(m))
	for k, v := range m {
		values = append(values, KV{k: k, v: v})
	}

	slices.SortFunc(values, func(a, b KV) int {
		return cmp.Compare(a.v, b.v)
	})

	if n > len(values) {
		n = len(values)
	}

	out := map[K]V{}
	for _, kv := range values[:n] {
		out[kv.k] = kv.v
	}

	return out
}

func clampAndFlip[K comparable, V float32 | float64](m map[K]V, clamp V) map[K]V {
	out := make(map[K]V, len(m))

	for k, v := range m {
		if v < 0 {
			out[k] = -v
		} else {
			out[k] = min(v, clamp)
		}
	}

	return out
}

func extractRecency(p *Persona, nodes []memory.NodeId) map[memory.NodeId]float64 {
	out := map[memory.NodeId]float64{}

	for i, node := range nodes {
		out[node] = math.Pow(p.state.RecencyDecay, float64(i+1))
	}

	return out
}

func extractImportance(p *Persona, nodes []memory.NodeId) map[memory.NodeId]float64 {
	out := map[memory.NodeId]float64{}

	for _, node := range nodes {
		out[node] = float64(p.associativeMemory.GetNode(node).Importance)
	}

	return out
}

func extractValence(p *Persona, nodes []memory.NodeId) map[memory.NodeId]float64 {
	out := map[memory.NodeId]float64{}

	for _, node := range nodes {
		out[node] = float64(p.associativeMemory.GetNode(node).Valence)
	}

	return out
}

func extractRelevance(p *Persona, nodes []memory.NodeId, focalPoint string) map[memory.NodeId]float64 {
	out := map[memory.NodeId]float64{}

	focalEmbedding := p.GetEmbedding(focalPoint)

	for _, node := range nodes {
		nodeEmbedding, _ := p.associativeMemory.GetEmbeddingByNodeId(node)
		out[node] = float64(cosineSimilarity(nodeEmbedding, focalEmbedding))
	}

	return out
}

type retrievalConfig struct {
	count int
}

type retrievalOpt func(*retrievalConfig)

func withRetrievalCount(n int) retrievalOpt {
	return func(rc *retrievalConfig) {
		rc.count = n
	}
}

func (p *Persona) retrieveForFocalPoints(focalPoints []string, retrievalOpts ...retrievalOpt) map[string][]memory.NodeId {
	config := retrievalConfig{
		count: 30,
	}

	retrieved := map[string][]memory.NodeId{}

	for _, focalPoint := range focalPoints {
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

		recencyScores := extractRecency(p, nodes)
		recencyScores = normalizeMap(recencyScores, 0, 1)
		importanceScores := extractImportance(p, nodes)
		importanceScores = normalizeMap(importanceScores, 0, 1)
		relevanceScores := extractRelevance(p, nodes, focalPoint)
		relevanceScores = normalizeMap(relevanceScores, 0, 1)
		valenceScores := extractValence(p, nodes)
		valenceScores = clampAndFlip(valenceScores, 0)
		valenceScores = normalizeMap(valenceScores, 0, 1)

		out := map[memory.NodeId]float64{}

		// NOTE(Friso): These weights come directly from the original code
		weights := struct{ recency, importance, relevance, valence float64 }{1, 1, 1, 1}

		for key := range recencyScores {
			out[key] = recencyScores[key]*weights.recency*p.state.RecencyWeight +
				importanceScores[key]*weights.importance*p.state.ImportanceWeight +
				relevanceScores[key]*weights.relevance*p.state.RelevanceWeight +
				valenceScores[key]*weights.valence*p.state.ValenceWeight
		}

		out = highestNValues(out, config.count)
		outNodes := make([]memory.NodeId, 0, len(out))
		for k := range out {
			p.associativeMemory.UpdateNode(k, func(c *memory.ConceptNode) {
				c.LastAccessed = p.state.CurrentTime
			})
			outNodes = append(outNodes, k)
		}

		if p.ctx.Log.Enabled(context.Background(), slog.LevelDebug) {
			logOut := make([]slog.Attr, 0, len(outNodes))

			for _, node := range outNodes {
				logOut = append(logOut, slog.Group(
					"node_"+strconv.Itoa(int(node)),
					slog.Int("id", int(node)),
					slog.Float64("final", out[node]),
					slog.Float64("recency", recencyScores[node]),
					slog.Float64("importance", importanceScores[node]),
					slog.Float64("valence", valenceScores[node]),
					slog.Float64("relevancy", relevanceScores[node]),
				),
				)
			}

			p.ctx.Log.Debug("retrieval",
				slog.String("type", "retrieval"),
				slog.String("retrieval_type", "focal_points"),
				slog.String("focal_point", focalPoint),
				slog.Int("count", config.count),
				slog.Group("retrieved", logOut),
			)
		}

		retrieved[focalPoint] = outNodes
	}

	return retrieved
}
