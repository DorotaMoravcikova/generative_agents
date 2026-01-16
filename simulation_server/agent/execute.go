package agent

import (
	"fmt"
	"math/rand"

	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

func sample[T any](arr []T, sampleSize int) []T {
	n := len(arr)

	rand.Shuffle(n, func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})

	if n < sampleSize {
		return arr
	}

	return arr[:sampleSize]
}

func (p *Persona) execute(m *maze.Maze, personas map[string]*Persona, plan memory.Path) (maze.TilePos, string, maze.Event) {
	if plan.HasState(memory.PathStateRandom) && len(p.state.PlannedPath) == 0 {
		p.state.ActivityPathSet = false
	}

	if !p.state.ActivityPathSet {
		targetTiles := []maze.TilePos{}

		if plan.HasState(memory.PathStatePersona) {
			targetPersonaPos := personas[plan.GetArg()].state.Position
			potentialPath := m.Pathfind(p.state.Position, targetPersonaPos)
			if len(potentialPath) <= 2 {
				targetTiles = []maze.TilePos{potentialPath[0]}
			} else {
				p1 := m.Pathfind(p.state.Position, potentialPath[len(potentialPath)/2])
				p2 := m.Pathfind(p.state.Position, potentialPath[len(potentialPath)/2+1])
				if len(p1) <= len(p2) {
					targetTiles = []maze.TilePos{potentialPath[len(potentialPath)/2]}
				} else {
					targetTiles = []maze.TilePos{potentialPath[len(potentialPath)/2+1]}
				}
			}
		} else if plan.HasState(memory.PathStateWaiting) {
			var x, y int
			n, err := fmt.Sscanf(plan.GetArg(), memory.WaitingArgFormat, &x, &y)
			if n != 2 {
				panic(fmt.Errorf("Parsed unexpected amount of wait argument, got: %d, expected 2", n))
			} else if err != nil {
				panic(fmt.Errorf("Could not parse waiting arguments: %w", err))
			}
			targetTiles = []maze.TilePos{{X: x, Y: y}}
		} else if plan.HasState(memory.PathStateRandom) {
			t, ok := m.PathToTiles(plan.AtLevel(memory.PathLevelArena))
			if !ok {
				panic(fmt.Errorf("could not find path in maze: %s", plan.ToString()))
			}
			targetTiles = sample(t, 1)
		} else {
			if t, ok := m.PathToTiles(plan); ok {
				targetTiles = t
			} else {
				// NOTE(Friso): This should probably not be a panic as its in the core simulation loop but idk what else to do now
				panic(fmt.Errorf("Path not present in maze: %s", plan.ToString()))
			}
		}

		targetTiles = sample(targetTiles, 4)

		newTargetTiles := []maze.TilePos{}
		for _, tile := range targetTiles {
			events := m.GetTile(tile).Events
			passTile := false
			for event := range events {
				if _, ok := personas[event.SPO.Subject]; ok {
					passTile = true
					break
				}
			}
			if !passTile {
				newTargetTiles = append(newTargetTiles, tile)
			}
		}
		if len(newTargetTiles) != 0 {
			targetTiles = newTargetTiles
		}

		currTile := p.state.Position
		closestTile := maze.TilePos{X: -1, Y: -1}
		path := []maze.TilePos{}

		for _, target := range targetTiles {
			currPath := m.Pathfind(currTile, target)
			if (closestTile == maze.TilePos{X: -1, Y: -1}) {
				closestTile = target
				path = currPath
			} else if len(currPath) < len(path) {
				closestTile = target
				path = currPath
			}
		}

		// The path returned by maze.Pathfind still includes the start tile, so skip that
		p.state.PlannedPath = path[1:]
		p.state.ActivityPathSet = true
	}

	tile := p.state.Position
	if len(p.state.PlannedPath) > 0 {
		tile = p.state.PlannedPath[0]
		p.state.PlannedPath = p.state.PlannedPath[1:]
	}

	description := fmt.Sprintf("%s @ %s", p.state.ActivityDescription, p.state.ActivityAddress.ToString())

	return tile, p.state.ActivityPronunciato, maze.Event{SPO: p.state.ActivitySPO, Description: description}
}
