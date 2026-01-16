package agent

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/llm"
	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

type SPODescription struct {
	Triple      memory.SPO
	Description string
}

type relevantNodes struct {
	currEvent memory.NodeId
	events    map[memory.NodeId]struct{}
	thoughts  map[memory.NodeId]struct{}
}

type State struct {
	// The current Position of the persona
	Position maze.TilePos
	// The current time of the persona, this may differ from the simulation time during simulation
	CurrentTime time.Time

	FullName  string
	FirstName string
	LastName  string
	Age       int
	// The innate traits of the persona
	InnateTraits  string
	LearnedTraits string
	// The current plans of the persona, corresponds to the "currently" field in the simulation json
	CurrentPlans string
	// The lifestyle of the persona
	Lifestyle  string
	LivingArea memory.Path

	// The vision radius of the persona
	VisionRadius int
	// The maximum number of events to consider during perception
	AttentionBandwidth int
	// The maximum number of events to consider during recall
	Retention int

	// The current reflection trigger value, if this goes below below 1 reflection occurs
	CurrentReflectionTrigger int
	// The value currentReflectionTrigger is reset to upon reflection
	ReflectionTrigger int
	// The amount of memories added since the last reflection
	ReflectionElements int

	// How fast the recency score of memories decays
	RecencyDecay float64

	// The requirements for the daily plan of the persona
	DailyPlanRequirements string
	// The persona's daily plan in broad strokes
	DailyPlan []string
	// The persona's daily schedule including all the activities they actually performed/will perform.
	DailySchedule []llm.Plan
	// The daily schedule as generated originally
	OriginalDailySchedule []llm.Plan

	// The path that the persona has planned to take.
	PlannedPath []maze.TilePos

	// The event triple the persona is currently engaged in
	ActivitySPO memory.SPO
	// A desctiption of the event the persona is currently engaged in
	ActivityDescription string
	ActivityPronunciato string
	// The location of where the current activity takes place
	ActivityAddress   memory.Path
	ActivityStartTime time.Time
	ActivityDuration  time.Duration
	// Whether the path for the current activity has been set
	ActivityPathSet bool

	// NOTE(Friso): I'm not sure why these fields are here, I cannot find any place where they are actually read.
	// But they are in the original code, so here I am.
	ActivityObjectDescription string
	ActivityObjectPronunciato string
	ActivityObjectSPO         memory.SPO

	// All the utterances said during the current Chat if there is one
	Chat []memory.Utterance
	// The end time of the current chat if there is one
	ChatEndTime time.Time
	// The name of the persona this persona is chatting with if there is one
	ChattingWith string
	// The amount of timesteps since we last initiated a conversationg with this Persona,
	// prevents Personas from engaging in endless loops of conversation.
	ChattingWithBuffer map[string]int

	RecencyWeight    float64
	ImportanceWeight float64
	RelevanceWeight  float64
	ValenceWeight    float64
}

func (s *State) SetActivity(plog *slog.Logger, activityAddress memory.Path, duration time.Duration, activityDescription string, activityPronunciato string, activitySPO memory.SPO, activityObjectDescription string, activityObjectPronunciato string, activityObjectSPO memory.SPO) {
	s.ActivityAddress = activityAddress
	s.ActivityDuration = duration
	s.ActivityDescription = activityDescription
	s.ActivityPronunciato = activityPronunciato
	s.ActivitySPO = activitySPO

	s.ChattingWith = ""
	s.Chat = []memory.Utterance{}
	s.ChatEndTime = time.Time{}

	s.ActivityObjectDescription = activityObjectDescription
	s.ActivityObjectPronunciato = activityObjectPronunciato
	s.ActivityObjectSPO = activityObjectSPO

	s.ActivityStartTime = s.CurrentTime
	s.ActivityPathSet = false

	plog.Info("set_activity",
		slog.String("type", "activity_set"),
		slog.String("node_type", "activity"),
		slog.String("address", activityAddress.ToString()),
		slog.String("start_time", s.CurrentTime.Format(time.RFC3339)),
		slog.Int("duration", int(duration.Minutes())),
	)
}

func (s *State) SetChatActivity(plog *slog.Logger, activityAddress memory.Path, duration time.Duration, activityDescription string, activityPronunciato string, activitySPO memory.SPO, chattingWith string, chat []memory.Utterance, chattingWithBuffer map[string]int, chatEndTime time.Time) {
	s.ActivityAddress = activityAddress
	s.ActivityDuration = duration
	s.ActivityDescription = activityDescription
	s.ActivityPronunciato = activityPronunciato
	s.ActivitySPO = activitySPO

	s.ChattingWith = chattingWith
	s.Chat = chat
	for k, v := range chattingWithBuffer {
		s.ChattingWithBuffer[k] = v
	}
	s.ChatEndTime = chatEndTime

	s.ActivityObjectDescription = ""
	s.ActivityObjectPronunciato = ""
	s.ActivityObjectSPO = memory.SPO{}

	s.ActivityStartTime = s.CurrentTime
	s.ActivityPathSet = false

	plog.Info("set_activity",
		slog.String("type", "activity_set"),
		slog.String("node_type", "chat"),
		slog.String("chatting_with", chattingWith),
		slog.String("address", activityAddress.ToString()),
		slog.String("start_time", s.CurrentTime.Format(time.RFC3339)),
		slog.Int("duration", int(duration.Minutes())),
	)
}

func (s State) IsActivityFinished() bool {
	if s.ActivityAddress.IsEmpty() {
		return true
	}

	var endTime time.Time
	if s.ChattingWith != "" {
		endTime = s.ChatEndTime
	} else {
		endTime = s.ActivityStartTime.Add(s.ActivityDuration)
	}

	return !s.CurrentTime.Before(endTime)
}

func (s State) GetDailyPlanIndex() int { return s.GetDailyPlanIndexInMinutes(0) }
func (s State) GetDailyPlanIndexInMinutes(advance int) int {
	timeElapsedToday := time.Duration(s.CurrentTime.Hour())*time.Hour +
		time.Duration(s.CurrentTime.Minute())*time.Minute +
		time.Duration(s.CurrentTime.Second())*time.Second +
		time.Duration(s.CurrentTime.Nanosecond()) +
		time.Duration(advance)*time.Minute

	currIndex := 0
	elapsed := time.Duration(0)
	for _, plan := range s.DailySchedule {
		elapsed += time.Duration(plan.Duration) * time.Minute
		if timeElapsedToday < elapsed {
			return currIndex
		}
		currIndex += 1
	}

	return currIndex
}

func (s State) GetOriginalDailyPlanIndex() int { return s.GetOriginalDailyPlanIndexInMinutes(0) }
func (s State) GetOriginalDailyPlanIndexInMinutes(advance int) int {
	timeElapsedToday := time.Duration(s.CurrentTime.Hour())*time.Hour +
		time.Duration(s.CurrentTime.Minute())*time.Minute +
		time.Duration(s.CurrentTime.Second())*time.Second +
		time.Duration(s.CurrentTime.Nanosecond()) +
		time.Duration(advance)*time.Minute

	currIndex := 0
	elapsed := time.Duration(0)
	for _, plan := range s.OriginalDailySchedule {
		elapsed += time.Duration(plan.Duration) * time.Minute
		if timeElapsedToday < elapsed {
			return currIndex
		}
		currIndex += 1
	}

	return currIndex
}

type Persona struct {
	name  string
	state State

	associativeMemory *memory.Associative
	spatialMemory     *memory.Spatial

	embedder  llm.Embedder
	cognition llm.Cognition

	// Context for the current move
	ctx MoveCtx
}

func (p *Persona) State() State {
	return p.state
}

func (p *Persona) Memory() (*memory.Associative, *memory.Spatial) {
	return p.associativeMemory, p.spatialMemory
}

func (p *Persona) ResetChattingWithBuffer() {
	p.state.ChattingWithBuffer = map[string]int{}
}

func New(name string, assocMem *memory.Associative, spatialMem *memory.Spatial, state State, embedder llm.Embedder, cognition llm.Cognition) *Persona {
	return &Persona{
		name:              name,
		associativeMemory: assocMem,
		spatialMemory:     spatialMem,
		state:             state,
		embedder:          embedder,
		cognition:         cognition,
	}
}

func (p *Persona) addChatToMemory(spo memory.SPO, description string, keywords []string, importance, valence int, chat []memory.Utterance, created time.Time, expiration *time.Time, embeddingKey string, embedding []float64) memory.ConceptNode {
	node := p.associativeMemory.AddChat(spo, description, keywords, importance, valence, chat, created, expiration, embeddingKey, embedding)
	p.ctx.Log.Info(
		"add_thought",
		slog.String("type", "memory_append"),
		slog.Int("node_id", int(node.Id)),
		slog.String("node_type", "chat"),
		slog.Int("importance", importance),
		slog.Int("valence", valence),
		slog.String("embedding_key", embeddingKey),
		slog.Any("expiration", expiration),
	)

	return node
}

func (p *Persona) addThoughtToMemory(spo memory.SPO, description string, keywords []string, importance, valence int, evidence []memory.NodeId, created time.Time, expiration *time.Time, embeddingKey string, embedding []float64) memory.ConceptNode {
	node := p.associativeMemory.AddThought(spo, description, keywords, importance, valence, evidence, created, expiration, embeddingKey, embedding)
	p.ctx.Log.Info(
		"add_thought",
		slog.String("type", "memory_append"),
		slog.Int("node_id", int(node.Id)),
		slog.String("node_type", "thought"),
		slog.Int("importance", importance),
		slog.Int("valence", valence),
		slog.String("embedding_key", embeddingKey),
		slog.Any("evidence", evidence),
		slog.Any("expiration", expiration),
	)

	return node
}

func (p *Persona) addEventToMemory(event maze.Event, keywords []string, importance, valence int, evidence []memory.NodeId, embeddingKey string, embedding []float64) memory.ConceptNode {
	node := p.associativeMemory.AddEvent(event.SPO, event.Description, keywords, importance, valence, evidence, p.state.CurrentTime, nil, embeddingKey, embedding)

	p.ctx.Log.Info(
		"add_event",
		slog.String("type", "memory_append"),
		slog.Int("node_id", int(node.Id)),
		slog.String("node_type", "event"),
		slog.Int("importance", importance),
		slog.Int("valence", valence),
		slog.String("embedding_key", embeddingKey),
		slog.Any("evidence", evidence),
	)

	p.state.CurrentReflectionTrigger -= importance
	p.state.ReflectionElements += 1

	return node
}

type MoveCtx struct {
	Log *slog.Logger
}

func (p *Persona) Move(ctx MoveCtx, maze *maze.Maze, personas map[string]*Persona, pos maze.TilePos, currTime time.Time) (next_tile maze.TilePos, pronunciato string, event maze.Event) {
	p.ctx = ctx
	p.ctx.Log = p.ctx.Log.With(slog.String("persona", p.name))

	start := time.Now()
	p.ctx.Log.Info("persona_step_start",
		slog.String("event", "persona_step_start"),
	)
	defer func() {
		p.ctx.Log.Info("persona_step_done",
			slog.String("event", "persona_step_done"),
			slog.Duration("duration", time.Since(start)),
		)
	}()

	newDay := NewDayTypeNoNewDay
	if p.state.CurrentTime.IsZero() {
		newDay = NewTypeDayFirstDay
	} else if isDifferentDate(p.state.CurrentTime, currTime) {
		newDay = NewDayTypeNewDay
	}
	p.state.CurrentTime = currTime

	phaseStart := time.Now()
	percieved := p.percieve(maze)
	p.ctx.Log.Debug("persona_phase_done",
		slog.String("event", "persona_phase_done"),
		slog.String("phase", "perceive"),
		slog.Duration("duration", time.Since(phaseStart)),
	)

	phaseStart = time.Now()
	retrieved := p.retrieveForPerceptions(percieved)
	p.ctx.Log.Debug("persona_phase_done",
		slog.String("event", "persona_phase_done"),
		slog.String("phase", "retrieve"),
		slog.Duration("duration", time.Since(phaseStart)),
	)

	phaseStart = time.Now()
	plan := p.plan(maze, personas, retrieved, newDay)
	p.ctx.Log.Debug("persona_phase_done",
		slog.String("event", "persona_phase_done"),
		slog.String("phase", "plan"),
		slog.Duration("duration", time.Since(phaseStart)),
	)

	phaseStart = time.Now()
	p.reflect()
	p.ctx.Log.Debug("persona_phase_done",
		slog.String("event", "persona_phase_done"),
		slog.String("phase", "reflect"),
		slog.Duration("duration", time.Since(phaseStart)),
	)

	defer func() {
		p.ctx.Log.Debug("persona_phase_done",
			slog.String("event", "persona_phase_done"),
			slog.String("phase", "execute"),
			slog.Duration("duration", time.Since(phaseStart)),
		)
	}()

	phaseStart = time.Now()
	return p.execute(maze, personas, plan)
}

func (p *Persona) GetCurrentEvent() maze.Event {
	if p.state.ActivityAddress.IsEmpty() {
		return maze.Event{SPO: memory.SPO{Subject: p.name}}
	} else {
		return maze.Event{SPO: p.state.ActivitySPO, Description: p.state.ActivityDescription}
	}
}

func (p *Persona) GetCurrentObjectEvent() maze.Event {
	if p.state.ActivityAddress.IsEmpty() {
		return maze.Event{}
	} else {
		return maze.Event{SPO: memory.SPO{
			Subject:   p.state.ActivityAddress.ToString(),
			Predicate: p.state.ActivityObjectSPO.Predicate,
			Object:    p.state.ActivityObjectSPO.Object,
		}, Description: p.state.ActivityObjectDescription}
	}
}

func (p *Persona) SetPosition(pos maze.TilePos) {
	p.state.Position = pos
}

func isDifferentDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()

	return ay != by || am != bm || ad != bd
}

func (p *Persona) GetChat() []memory.Utterance {
	return p.state.Chat
}

// CurrentTime implements llm.Persona.
func (p *Persona) CurrentTime() time.Time {
	return p.state.CurrentTime
}

// DailyPlan implements llm.Persona.
func (p *Persona) DailyPlan() []string {
	return p.state.DailyPlan
}

// IdentityStableSet implements llm.Persona.
func (p *Persona) IdentityStableSet() string {
	traits := []string{}
	traits = append(traits, fmt.Sprintf("Name: %s", p.name))
	traits = append(traits, fmt.Sprintf("Age: %d", p.state.Age))
	traits = append(traits, fmt.Sprintf("Innate traits: %s", p.state.InnateTraits))
	traits = append(traits, fmt.Sprintf("Learned traits: %s", p.state.LearnedTraits))
	traits = append(traits, fmt.Sprintf("Currently: %s", p.state.CurrentPlans))
	traits = append(traits, fmt.Sprintf("Lifestyle: %s", p.state.Lifestyle))
	traits = append(traits, fmt.Sprintf("Daily plan requirements: %s", p.state.DailyPlanRequirements))
	traits = append(traits, fmt.Sprintf("Current date: %s", p.name))

	return strings.Join(traits, ",\n")
}

// Lifestyle implements llm.Persona.
func (p *Persona) Lifestyle() string {
	return p.state.Lifestyle
}

// Name implements llm.Persona.
func (p *Persona) Name() string {
	return p.name
}

// OriginalHourlySchedule implements llm.Persona.
func (p *Persona) OriginalHourlySchedule() []llm.Plan {
	return p.state.OriginalDailySchedule
}

// OriginalHourlyScheduleIndex implements llm.Persona.
func (p *Persona) OriginalHourlyScheduleIndex() int {
	return p.state.GetOriginalDailyPlanIndex()
}

func (p *Persona) LivingArea() memory.Path {
	return p.state.LivingArea
}

func quote(strs []string) []string {
	out := make([]string, 0, len(strs))

	for _, str := range strs {
		out = append(out, "\""+str+"\"")
	}

	return out
}

// KnownArenas implements llm.Persona.
func (p *Persona) KnownArenas(path memory.Path) []string {
	return quote(p.spatialMemory.GetKnown(path, memory.PathLevelArena))
}

func (p *Persona) KnownSectors(path memory.Path) []string {
	return quote(p.spatialMemory.GetKnown(path, memory.PathLevelSector))
}

func (p *Persona) KnownObjects(path memory.Path) []string {
	return quote(p.spatialMemory.GetKnown(path, memory.PathLevelObject))
}

// Position implements llm.Persona.
func (p *Persona) Position() maze.TilePos {
	return p.state.Position
}

func (p *Persona) DailyPlanRequirements() string {
	return p.state.DailyPlanRequirements
}

func (p *Persona) CurrentPlans() string {
	return p.state.CurrentPlans
}

func (p *Persona) CurrentChat() []memory.Utterance {
	return p.state.Chat
}

func (p *Persona) DailySchedule() []llm.Plan {
	return p.state.DailySchedule
}

func (p *Persona) DailyScheduleIdx() int {
	return p.state.GetDailyPlanIndex()
}

func (p *Persona) StartOfDay() time.Time {
	return time.Date(p.CurrentTime().Year(), p.CurrentTime().Month(), p.CurrentTime().Day(), 0, 0, 0, 0, p.CurrentTime().Location())
}

func (p *Persona) WakeUpTime() (t time.Time) {
	if !strings.Contains(p.DailySchedule()[p.DailyScheduleIdx()].Activity, "sleeping") {
		return
	}

	minutesUntilWakeUp := 0
	for i := 0; i <= p.DailyScheduleIdx(); i += 1 {
		minutesUntilWakeUp += p.DailySchedule()[i].Duration
	}

	return p.StartOfDay().Add(time.Duration(minutesUntilWakeUp) * time.Minute)
}

func (p *Persona) PlannedPath() []maze.TilePos {
	return p.state.PlannedPath
}

func (p *Persona) LastChat(other string) (memory.NodeId, bool) {
	return p.associativeMemory.GetLastChat(other)
}

func (p *Persona) GetMemory(node memory.NodeId) memory.ConceptNode {
	return p.associativeMemory.GetNode(node)
}

func (p *Persona) ActivityDescription() string {
	return p.state.ActivityDescription
}

func (p *Persona) ActivityEndTime(idx int) time.Time {
	t := p.StartOfDay()

	for i := 0; i < idx; i += 1 {
		t = t.Add(time.Duration(p.DailySchedule()[i].Duration) * time.Minute)
	}

	return t
}

func (p *Persona) GetEmbedding(str string) []float64 {
	embedding, ok := p.associativeMemory.GetEmbedding(str)
	if !ok {
		embedding = p.embedder.GenerateEmbedding(str)
		p.associativeMemory.SaveEmbedding(str, embedding)
	}

	return embedding
}
