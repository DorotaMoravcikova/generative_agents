package server

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/agent"
	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

type SimulationStorer interface {
	SaveMovements(step int, movements map[string]PersonaMovement, currTime time.Time) error
	SaveSimulation(srv *Server) error
	Backup(step int) error
}

type Server struct {
	CurrentTime time.Time
	StartTime   time.Time
	// How much time the simulation progresses each step
	TimeStep time.Duration
	Maze     *maze.Maze
	// The step the current simulation is on
	Step             int
	Personas         map[string]*agent.Persona
	PersonaPositions map[string]maze.TilePos
	ForkedSim        string
	// After how many steps we make a backup of the simulation state
	BackupInterval int

	Log *slog.Logger

	Storage SimulationStorer
}

func New() *Server {
	return &Server{}
}

type PersonaMovement struct {
	Tile        maze.TilePos
	Pronunciato string
	Event       maze.Event
	Chat        []memory.Utterance
}

type Movements struct {
	Personas    map[string]PersonaMovement
	CurrentTime time.Time
}

func (s *Server) Run(i int) error {
	for range i {
		if s.Step%s.BackupInterval == 0 {
			if err := s.Storage.Backup(s.Step); err != nil {
				return fmt.Errorf("could not create server backup: %w", err)
			}
		}

		s.ExecuteStep()
		if err := s.Storage.SaveSimulation(s); err != nil {
			return fmt.Errorf("could not save simulation: %w", err)
		}
	}

	return nil
}

func (s *Server) ExecuteStep() {
	stepLog := s.Log.With(
		slog.Int("step", s.Step),
		slog.String("type", "step"),
		slog.Time("sim_time", s.CurrentTime),
	)

	stepLog.Info("step_start", slog.String("phase", "start"))

	s.skipSleep(stepLog)

	gameObjectCleanup := map[maze.Event]maze.TilePos{}
	movements := Movements{Personas: map[string]PersonaMovement{}, CurrentTime: s.CurrentTime}

	// If the persona is at their destination activate their object event
	for _, persona := range s.Personas {
		if len(persona.PlannedPath()) != 0 {
			continue
		}

		ev := persona.GetCurrentObjectEvent()
		if ev.SPO.Subject == "" {
			continue
		}

		gameObjectCleanup[ev] = persona.Position()
		s.Maze.AddEventToTile(persona.Position(), ev)
		s.Maze.RemoveEventFromTile(persona.Position(), maze.Event{SPO: memory.SPO{Subject: ev.SPO.Subject}})
	}

	for name, persona := range s.Personas {
		ctx := agent.MoveCtx{
			Log: stepLog,
		}
		next, pronunciato, event := persona.Move(ctx, s.Maze, s.Personas, s.PersonaPositions[name], s.CurrentTime)

		movements.Personas[name] = PersonaMovement{
			Tile:        next,
			Pronunciato: pronunciato,
			Event:       event,
			Chat:        persona.GetChat(),
		}
	}

	for name, persona := range s.Personas {
		curr := persona.Position()
		next := movements.Personas[name].Tile

		s.Maze.RemoveSubjectEventsFromTile(curr, name)
		s.Maze.AddEventToTile(next, persona.GetCurrentEvent())
	}

	for name, movement := range movements.Personas {
		s.PersonaPositions[name] = movement.Tile
		s.Personas[name].SetPosition(movement.Tile)
	}

	for ev, pos := range gameObjectCleanup {
		s.Maze.TurnTileEventIdle(pos, ev)
	}

	if err := s.Storage.SaveMovements(s.Step, movements.Personas, movements.CurrentTime); err != nil {
		panic(fmt.Sprintf("Could not save movements: %v", err))
	}

	stepLog.Info("step_end",
		slog.String("phase", "end"),
	)

	s.CurrentTime = s.CurrentTime.Add(s.TimeStep)
	s.Step += 1

	// TODO(Friso): Actually saving the updated state
}

func (s *Server) skipSleep(stepLog *slog.Logger) {
	step := 3

	midnight := time.Date(
		s.CurrentTime.Year(),
		s.CurrentTime.Month(),
		s.CurrentTime.Day(),
		0, 0, 0, 0,
		s.CurrentTime.Location(),
	)

	elapsed := s.CurrentTime.Sub(midnight)
	iterationsSinceDay := int(elapsed / s.TimeStep)

	// NOTE(Friso): We don't skip the first few iterations of the day as that is when we plan the daily schedule
	if iterationsSinceDay < step {
		return
	}

	var earliestWakeUpTime time.Time
	for _, p := range s.Personas {
		if !strings.Contains(p.DailySchedule()[p.DailyScheduleIdx()].Activity, "sleeping") {
			// NOTE(Friso): This means the agent is currently not asleep, we can only skip if all agents are sleeping
			return
		}

		t := p.WakeUpTime()
		if t.IsZero() || !p.StartOfDay().Before(t.Add(-s.TimeStep*time.Duration(step))) {
			return
		}
		t = t.Add(-s.TimeStep * time.Duration(step))

		if earliestWakeUpTime.IsZero() {
			earliestWakeUpTime = t
		} else if t.Before(earliestWakeUpTime) {
			earliestWakeUpTime = t
		}
	}

	if !s.CurrentTime.Before(earliestWakeUpTime) {
		// NOTE(Friso): Just to be safe we have them complete one last timestep whilst sleeping
		// This ensures we don't accidentally go back in time
		return
	}

	stepLog.With(slog.String("type", "skip_sleep"), slog.Time("next_step_time", earliestWakeUpTime)).Debug("skipping sleep")

	s.CurrentTime = earliestWakeUpTime
	for _, p := range s.Personas {
		// NOTE(Friso): Since we actually skipped sleep, we need to ensure that all time dependent state in the agents matches the expected values.
		p.ResetChattingWithBuffer()
	}
}
