package llm

import (
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/maze"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

type Embedder interface {
	GenerateEmbedding(string) []float64
}

type Persona interface {
	Name() string
	LivingArea() memory.Path

	Lifestyle() string
	CurrentPlans() string
	IdentityStableSet() string

	CurrentTime() time.Time
	StartOfDay() time.Time

	CurrentChat() []memory.Utterance
	LastChat(name string) (memory.NodeId, bool)

	DailyPlanRequirements() string
	DailyPlan() []string
	DailySchedule() []Plan
	DailyScheduleIdx() int
	OriginalHourlySchedule() []Plan
	OriginalHourlyScheduleIndex() int

	ActivityDescription() string
	ActivityEndTime(idx int) time.Time

	KnownSectors(memory.Path) []string
	KnownArenas(memory.Path) []string
	KnownObjects(memory.Path) []string

	GetMemory(memory.NodeId) memory.ConceptNode

	Position() maze.TilePos
	PlannedPath() []maze.TilePos
}

type Maze interface {
	GetTile(maze.TilePos) maze.Tile
	Exists(memory.Path) bool
}

type Plan struct {
	// The activity that the agent should perform.
	Activity string
	// The duration of the activity in minutes.
	Duration int
}

type Cognition interface {
	// Generates an importance score for a memory of a specific type based off of the
	// persona's personality and the event description.
	GenerateImportanceScore(p Persona, nt memory.NodeType, description string) int
	// Generates an importance score for a chat based off of the description, transcript and the persona's personality
	GenerateImportanceScoreChat(p Persona, transcript []memory.Utterance, description string) int
	// Generates a valence score for a memory of an agent
	GenerateValenceScore(p Persona, nt memory.NodeType, description string) int
	// Generates a valnce score for a chat based off of the description, transcript and the persona's personality
	GenerateValenceScoreChat(p Persona, transcript []memory.Utterance, description string) int

	// Generates the wake up hour for the next day based off of the persona's personality.
	GenerateWakeUpHour(p Persona) time.Time
	// Generates the first daily plan for a persona.
	GenerateDailyPlan(p Persona, wakeUpHour time.Time) []string
	// Generates an hour schedule for a new day.
	GenerateHourlySchedule(p Persona, wakeUpHour time.Time) []Plan

	// Generates a list of sub-plans that the given plan should consist of
	GeneratePlanDecomposition(p Persona, plan Plan) []Plan

	// Generates an updated schedule in response to an event
	GenerateReactionScheduleUpdate(p Persona, insertedActivity Plan, startTime, endTime time.Time) []Plan

	// Generates the sector an activity should take place in
	GenerateActivitySector(p Persona, maze Maze, activity string, world string) string
	// Generates the arena an activity should take place in
	GenerateActivityArena(p Persona, maze Maze, activity string, world string, sector string) string
	// Generates the object that should be used for an activity
	GenerateActivityObject(p Persona, maze Maze, activity string, path memory.Path) string
	// Generates a pronunciato (2 emojis) representing the current activity taking place
	GenerateActivityPronunciato(p Persona, activity string) string
	// Generates a SPO (activity subject-predicate-object) triple
	GenerateActivitySPO(p Persona, activity string) memory.SPO

	// Generates a description for the object that is used in the current activity
	GenerateActivityObjectDescription(p Persona, object string, activity string) string
	// Generates a pronunciato (2 emojis) representing t for the object that is used in the current activity
	GenerateActivityObjectPronunciato(p Persona, activityObjectDescription string) string
	// Generates a SPO (activity subject-predicate-object) triple
	GenerateActivityObjectSPO(p Persona, object string, activityObjectDescription string) memory.SPO

	// Generates whether Persona init wants to talk to persona target
	GenerateDecideToTalk(init, target Persona, events, thoughts []memory.NodeId) bool
	// Generates whether init should wait until target has finished their activity before approaching them,
	// or init should continue with their own activity.
	// NOTE(Friso): In the original code this is called generate_decide_to_react, but this name is more apt.
	GenerateDecideToWait(init, target Persona, events, thoughts []memory.NodeId) (wait bool)

	// Generates a summary for a conversation that a persona had
	GenerateConversationSummary(p Persona, conversation []memory.Utterance) string
	// Generates a change in planning for p that should be remembered based off of a conversation
	GeneratePlanningThoughtAfterConversation(p Persona, conversation []memory.Utterance) string
	// Generates anything noteworthy that should be remembered after a conversation
	GenerateMemoAfterConversation(p Persona, conversation []memory.Utterance) string
	// Generates a summary of a relationship between init and target given the memories that init has of target
	GenerateRelationshipSummary(init, target Persona, memories []memory.NodeId) string
	// Generates one utterance in a conversation
	GenerateOneUtterance(init, target Persona, maze Maze, currentChat []memory.Utterance, relevant []memory.NodeId, relationship string) (utt memory.Utterance, endConversation bool)

	// Generates a list of focal points to address during reflection
	GenerateFocalPoints(p Persona, statements []memory.NodeId, numFocalPoints int) []string
	// Generates insights based off of the evidence presented in nodes
	GenerateInsightAndEvidence(p Persona, nodes []memory.NodeId, insightCount int) map[string][]memory.NodeId

	// Generates information the agent should remember when planning for the next day
	GeneratePlanningNote(p Persona, statements []string) string
	// Generates the feelings an agent has about their days up till now
	GeneratePlanningFeelings(p Persona, statements []string) string
	// Generates a new set of plans for an agent
	GenerateCurrentPlans(p Persona, plans, thoughts string) string
	// Generates new daily requirements
	GenerateNewDailyRequirements(p Persona) string
}
