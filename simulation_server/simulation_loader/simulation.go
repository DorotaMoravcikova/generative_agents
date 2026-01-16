package simulationloader

// NOTE(Friso): This entire package is a mess, but sunken cost fallacy I guess
// Also no matter how you rewrite it you will need to deal with Park's python-ness

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/agent"
	"github.com/fvdveen/generative_agents/simulation_server/llm"
	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/server"
)

func LoadSimulation(simulationPath string, mazeFolder string, embedder llm.Embedder, cognition llm.Cognition, logger *slog.Logger) (*server.Server, error) {
	content, err := os.ReadFile(path.Join(simulationPath, "reverie", "meta.json"))
	if err != nil {
		return nil, fmt.Errorf("could not read simulation meta file: %w", err)
	}

	var meta SimulationMeta
	if err = json.Unmarshal(content, &meta); err != nil {
		return nil, fmt.Errorf("could not unmarshal meta file json: %w", err)
	}

	m, err := LoadMaze(path.Join(mazeFolder, meta.MazeName), meta.MazeName)
	if err != nil {
		return nil, fmt.Errorf("could not load maze: %w", err)
	}

	content, err = os.ReadFile(path.Join(simulationPath, "environment", fmt.Sprintf("%d.json", meta.Step)))
	if err != nil {
		return nil, fmt.Errorf("could not read simulation environment file: %w", err)
	}

	var env Environment
	if err = json.Unmarshal(content, &env); err != nil {
		return nil, fmt.Errorf("could not unmarshal environment file: %w", err)
	}

	personas := map[string]*agent.Persona{}
	personaTiles := map[string]maze.TilePos{}
	for _, name := range meta.PersonaNames {
		envPersona, ok := env.Personas[name]
		if !ok {
			return nil, fmt.Errorf("persona missing from environment file: %s", name)
		}

		pos := maze.TilePos{X: envPersona.X, Y: envPersona.Y}
		p, err := LoadPersona(path.Join(simulationPath, "personas", name), pos, embedder, cognition, logger)
		if err != nil {
			return nil, fmt.Errorf("could not load persona %s: %w", name, err)
		}

		personas[name] = p
		personaTiles[name] = pos
		p.SetPosition(pos)
		m.AddEventToTile(pos, p.GetCurrentEvent())
	}

	s := server.New()

	s.CurrentTime = time.Time(meta.CurrTime)
	s.StartTime = time.Time(meta.StartDate)
	s.TimeStep = time.Duration(meta.SecondsPerStep) * time.Second
	s.Maze = m
	s.Step = meta.Step
	s.Personas = personas
	s.PersonaPositions = personaTiles
	s.ForkedSim = meta.ForkSimCode
	s.BackupInterval = meta.BackupInterval
	s.Log = logger

	s.Log.Debug("simulation loaded successfully")

	return s, nil
}
