package simulationloader

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
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

func LoadPersona(folder string, position maze.TilePos, embedder llm.Embedder, cognition llm.Cognition, log *slog.Logger) (*agent.Persona, error) {
	folder = path.Join(folder, "bootstrap_memory")

	assocMem, err := LoadAssociativeMemory(path.Join(folder, "associative_memory"))
	if err != nil {
		return nil, fmt.Errorf("could not load associative memory: %w", err)
	}

	spatialMem, err := LoadSpatialMemory(path.Join(folder, "spatial_memory.json"))
	if err != nil {
		return nil, fmt.Errorf("could not load spatial memory: %w", err)
	}

	state, err := LoadState(path.Join(folder, "scratch.json"), position)
	if err != nil {
		return nil, fmt.Errorf("could not load state: %w", err)
	}

	return agent.New(state.FullName, assocMem, spatialMem, *state, embedder, cognition), nil
}

func LoadState(stateFile string, position maze.TilePos) (*agent.State, error) {
	content, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("could not read state file: %w", err)
	}

	var state PersonaState
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, fmt.Errorf("could not unmarshal state json: %w", err)
	}

	schedule := make([]llm.Plan, 0, len(state.FDailySchedule))
	originalSchedule := make([]llm.Plan, 0, len(state.FDailyScheduleHourlyOrg))

	for _, plan := range state.FDailySchedule {
		schedule = append(schedule, llm.Plan{
			Activity: plan.Activity,
			Duration: plan.Duration,
		})
	}
	for _, plan := range state.FDailyScheduleHourlyOrg {
		originalSchedule = append(originalSchedule, llm.Plan{
			Activity: plan.Activity,
			Duration: plan.Duration,
		})
	}

	plannedPath := make([]maze.TilePos, 0, len(state.PlannedPath))
	for _, p := range state.PlannedPath {
		plannedPath = append(plannedPath, maze.TilePos{X: p.X, Y: p.Y})
	}

	chat := make([]memory.Utterance, 0, len(state.Chat))
	for _, u := range state.Chat {
		chat = append(chat, memory.Utterance{
			Speaker:  u.Speaker,
			Sentence: u.Utterance,
		})
	}

	endTime := time.Time{}
	if state.ChattingEndTime != nil {
		endTime = *(*time.Time)(state.ChattingEndTime)
	}

	chattingWith := ""
	if state.ChattingWith != nil {
		chattingWith = *state.ChattingWith
	}

	s := &agent.State{
		Position:                 position,
		CurrentTime:              time.Time(state.CurrTime),
		VisionRadius:             state.VisionR,
		AttentionBandwidth:       state.AttBandwidth,
		Retention:                state.Retention,
		CurrentReflectionTrigger: state.ImportanceTriggerCurr,
		ReflectionTrigger:        state.ImportanceTriggerMax,
		ReflectionElements:       state.ImportanceEleN,
		RecencyDecay:             state.RecencyDecay,
		DailyPlanRequirements:    state.DailyPlanReq,
		DailyPlan:                state.DailyReq,
		DailySchedule:            schedule,
		OriginalDailySchedule:    originalSchedule,
		PlannedPath:              plannedPath,
		ActivitySPO: memory.SPO{
			Subject:   state.ActEvent.Subject,
			Predicate: state.ActEvent.Predicate,
			Object:    state.ActEvent.Object,
		},
		ActivityDescription:       state.ActDescription,
		ActivityPronunciato:       state.ActPronunciatio,
		ActivityAddress:           memory.ParsePath(state.ActAddress),
		ActivityStartTime:         time.Time(state.ActStartTime),
		ActivityDuration:          time.Duration(state.ActDuration) * time.Minute,
		ActivityPathSet:           state.ActPathSet,
		ActivityObjectDescription: state.ActObjDescription,
		ActivityObjectPronunciato: state.ActObjPronunciatio,
		ActivityObjectSPO: memory.SPO{
			Subject:   state.ActObjEvent.Subject,
			Predicate: state.ActObjEvent.Predicate,
			Object:    state.ActObjEvent.Object,
		},
		Chat:               chat,
		ChatEndTime:        endTime,
		ChattingWith:       chattingWith,
		ChattingWithBuffer: state.ChattingWithBuffer,
		RecencyWeight:      state.RecencyW,
		ImportanceWeight:   state.ImportanceW,
		RelevanceWeight:    state.RecencyW,
		ValenceWeight:      state.ValenceW,
		FirstName:          state.FirstName,
		LastName:           state.LastName,
		Age:                state.Age,
		InnateTraits:       state.Innate,
		LearnedTraits:      state.Learned,
		CurrentPlans:       state.Currently,
		Lifestyle:          state.Lifestyle,
		LivingArea:         memory.ParsePath(state.LivingArea),
		FullName:           state.Name,
	}

	return s, nil
}
