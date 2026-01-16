package openai

import (
	"github.com/fvdveen/generative_agents/simulation_server/llm"
	"github.com/fvdveen/generative_agents/simulation_server/memory"
)

type WakeUpHourV2Input struct {
	Persona llm.Persona
}

type GeneratePoignancyThoughtV1Input struct {
	Persona llm.Persona
	Thought string
}

type GeneratePoignancyEventV1Input struct {
	Persona llm.Persona
	Event   string
}

type GeneratePoignancyChatV1Input struct {
	Persona      llm.Persona
	Description  string
	Conversation []memory.Utterance
}

type GenerateValenceThoughtV1Input struct {
	Persona llm.Persona
	Thought string
}

type GenerateValenceEventV1Input struct {
	Persona llm.Persona
	Event   string
}

type GenerateValenceChatV1Input struct {
	Persona      llm.Persona
	Description  string
	Conversation []memory.Utterance
}

type DailyPlanningV7Input struct {
	Persona     llm.Persona
	WakeUpHour  string
	CurrentDate string
}

type GenerateHourlyScheduleV2Input struct {
	Persona llm.Persona
}

type NewDecompScheduleV2Input struct {
	Persona                       llm.Persona
	OriginalStartTime             string
	OriginalEndTime               string
	PlanningFromTime              string
	OriginalPlans, TruncatedPlans []struct {
		StartTime string
		EndTime   string
		Activity  string
	}
	Inserted llm.Plan
}

type TaskDecompV3Input struct {
	Persona llm.Persona

	CurrentDate string

	Activity          string
	ActivityStartTime string
	ActivityEndTime   string
	ActivityDuration  int

	Schedule []struct {
		Activity  string
		StartTime string
		EndTime   string
	}
}

type ActionLocationSectorV3Input struct {
	Persona         llm.Persona
	CurrentLocation memory.Path

	Action    string
	SubAction string
}

type ActionLocationArenaV1Input struct {
	Persona         llm.Persona
	CurrentLocation memory.Path
	TargetLocation  memory.Path

	Activity          string
	SubAction         string
	DestinationSector string
}

type ActionObjectV1Input struct {
	Persona        llm.Persona
	TargetLocation memory.Path
	Activity       string
}

type GeneratePronunciatioV2Input struct {
	Activity string
}

type GenerateEventTripleV2Input struct {
	Name     string
	Activity string
}

type GenerateObjEventV2Input struct {
	Persona  llm.Persona
	Object   string
	Activity string
}

type DecideToTalkV3Input struct {
	Initiator, Target             llm.Persona
	InitiatorStatus, TargetStatus string
	Context                       string
	CurrentTime                   string
	LastChatTime                  string
	LastChatTopic                 string
}

type DecideToReactV2Input struct {
	Initiator, Target             llm.Persona
	InitiatorStatus, TargetStatus string
	Context                       string
	CurrentTime                   string
	TargetEndTime                 string
}

type IterativeConvoV2Input struct {
	Init, Target        llm.Persona
	Relevant            []memory.NodeId
	CurrentLocation     string
	RelationshipSummary string
	Conversation        []memory.Utterance
}

type SummarizeChatRelationshipV2Input struct {
	Init, Target llm.Persona
	Memories     []memory.NodeId
}

type SummarizeConversationV2Input struct {
	Conversation []memory.Utterance
}

type PlanningThoughtOnConvoV2Input struct {
	Persona      llm.Persona
	Conversation []memory.Utterance
}

type MemoOnConvoV1Input struct {
	Persona      llm.Persona
	Conversation []memory.Utterance
}

type GenerateFocalPtV2Input struct {
	Persona    llm.Persona
	Statements []memory.NodeId
	Count      int
}

type InsightAndEvidenceV2Input struct {
	Persona    llm.Persona
	Statements []memory.NodeId
	Count      int
}

type DescribeAgentFeelingsV1Input struct {
	Persona    llm.Persona
	Statements []string
}

type ExtractSchedulingInformationV1Input struct {
	Persona     llm.Persona
	Statements  []string
	CurrentDate string
}

type GenerateCurrentlyV1Input struct {
	Persona                    llm.Persona
	CurrentDate, YesterdayDate string
	PlanningNote, ThoughtNote  string
}

type ReviseDailyRequirementsV1Input struct {
	Persona     llm.Persona
	CurrentDate string
}

// Output structs for prompts
// ActionLocationObjectV2Output represents the output for ActionLocationObjectV2 prompt
type ActionLocationObjectV2Output struct {
	Output string `json:"output"`
}

// ActionLocationSectorV3Output represents the output for ActionLocationSectorV3 prompt
type ActionLocationSectorV3Output struct {
	Output string `json:"output"`
}

// ActionObjectV3Output represents the output for ActionObjectV3 prompt
type ActionObjectV1Output struct {
	Output string `json:"output"`
}

// AgentChatV1Output represents the output for AgentChatV1 prompt
type AgentChatV1Output struct {
	Utterance string `json:"utterance"`
}

// BedHourV1Output represents the output for BedHourV1 prompt
type BedHourV1Output struct {
	BedTime string `json:"bed_time"`
}

// ConvoToThoughtsV1Output represents the output for ConvoToThoughtsV1 prompt
type ConvoToThoughtsV1Output struct {
	Thought string `json:"thought"`
}

// CreateConversationV1Output represents the output for CreateConversationV1 prompt
type CreateConversationV1Output struct {
	Dialogue string `json:"dialogue"`
}

// CreateConversationV2Output represents the output for CreateConversationV2 prompt
type CreateConversationV2Output struct {
	Dialogue string `json:"dialogue"`
}

// DailyPlanningV7Output represents the output for DailyPlanningV7 prompt
type DailyPlanningV7Output struct {
	Schedule []string `json:"schedule"`
}

// DecideToReactV1Output represents the output for DecideToReactV1 prompt
type DecideToReactV1Output struct {
	Reasoning string `json:"reasoning"`
	Choice    string `json:"choice"`
}

// DecideToReactV2Output represents the output for DecideToReactV2 prompt
type DecideToReactV2Output struct {
	Reasoning string `json:"reasoning"`
	Choice    int    `json:"choice"`
}

// DecideToTalkV3Output represents the output for DecideToTalkV3 prompt
type DecideToTalkV3Output struct {
	Context    string `json:"context"`
	Question   string `json:"question"`
	Reasoning  string `json:"reasoning"`
	ShouldTalk string `json:"should_talk"`
}

// GenerateEventTripleV2Output represents the output for GenerateEventTripleV2 prompt
type GenerateEventTripleV2Output struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

// GenerateFocalPtV2Output represents the output for GenerateFocalPtV2 prompt
type GenerateFocalPtV2Output struct {
	FocalPoints []string `json:"questions"`
}

// GenerateHourlyScheduleV1Output represents the output for GenerateHourlyScheduleV1 prompt
type GenerateHourlyScheduleV1Output struct {
	Schedule [][]string `json:"schedule"`
}

// GenerateHourlyScheduleV2Output represents the output for GenerateHourlyScheduleV2 prompt
type GenerateHourlyScheduleV2Output struct {
	Schedule []struct {
		Time     string `json:"time"`
		Activity string `json:"activity"`
	} `json:"schedule"`
}

// GenerateNextConvoLineV1Output represents the output for GenerateNextConvoLineV1 prompt
type GenerateNextConvoLineV1Output struct {
	Utterance string `json:"utterance"`
}

// GenerateObjEventV2Output represents the output for GenerateObjEventV2 prompt
type GenerateObjEventV2Output struct {
	State string `json:"state"`
}

// GeneratePronunciatioV2Output represents the output for GeneratePronunciatioV2 prompt
type GeneratePronunciatioV2Output struct {
	Emoji string `json:"emoji"`
}

// GetKeywordsV1Output represents the output for GetKeywordsV1 prompt
type GetKeywordsV1Output struct {
	Factual []string `json:"factual"`
	Emotive []string `json:"emotive"`
}

// InsightAndEvidenceV2Output represents the output for InsightAndEvidenceV2 prompt
type InsightAndEvidenceV2Output struct {
	Insights []struct {
		Insight string `json:"insight"`
		Reasons []int  `json:"reasons"`
	} `json:"insights"`
}

// InterpretDayV1Output represents the output for InterpretDayV1 prompt
type InterpretDayV1Output struct {
	FeelingsAboutToday string `json:"feelings_about_today"`
	WantsTomorrow      string `json:"wants_for_tomorrow"`
}

// IterativeConvoV2Output represents the output for IterativeConvoV2 prompt
type IterativeConvoV2Output struct {
	Utterance        string `json:"utterance"`
	EndsConversation bool   `json:"ends_conversation"`
}

// KeywordToThoughtsV1Output represents the output for KeywordToThoughtsV1 prompt
type KeywordToThoughtsV1Output struct {
	Thought string `json:"thought"`
}

// MemoOnConvoV1Output represents the output for MemoOnConvoV1 prompt
type MemoOnConvoV1Output struct {
	Memo string `json:"memo"`
}

// NewDailyPlanV1Output represents the output for NewDailyPlanV1 prompt
type NewDailyPlanV1Output []string

// NewDecompScheduleV2Output represents the output for NewDecompScheduleV2 prompt
type NewDecompScheduleV2Output struct {
	Schedule []struct {
		StartTime       string `json:"start_time"`
		EndTime         string `json:"end_time"`
		Action          string `json:"action"`
		DurationMinutes int    `json:"duration_in_minutes"`
	} `json:"schedule"`
}

// NextStepSchedulingV2Output represents the output for NextStepSchedulingV2 prompt
type NextStepSchedulingV2Output struct {
	Action          string `json:"action"`
	DurationMinutes int    `json:"duration_minutes"`
}

// NextStepSchedullingV3Output represents the output for NextStepSchedullingV3 prompt
type NextStepSchedullingV3Output struct {
	Action          string `json:"action"`
	DurationMinutes int    `json:"duration_minutes"`
}

// PlanningThoughtOnConvoV2Output represents the output for PlanningThoughtOnConvoV2 prompt
type PlanningThoughtOnConvoV2Output struct {
	PlanningThought string `json:"planning_thought"`
}

// PoignancyChatV1Output represents the output for PoignancyChatV1 prompt
type PoignancyChatV1Output struct {
	Reasoning string `json:"reasoning"`
	Poignancy int    `json:"poignancy"`
}

// PoignancyEventV1Output represents the output for PoignancyEventV1 prompt
type PoignancyEventV1Output struct {
	Reasoning string `json:"reasoning"`
	Poignancy int    `json:"poignancy"`
}

// PoignancyThoughtV1Output represents the output for PoignancyThoughtV1 prompt
type PoignancyThoughtV1Output struct {
	Reasoning string `json:"reasoning"`
	Poignancy int    `json:"poignancy"`
}

// SummarizeChatIdeasV1Output represents the output for SummarizeChatIdeasV1 prompt
type SummarizeChatIdeasV1Output struct {
	Summary string `json:"summary"`
}

// SummarizeChatRelationshipV1Output represents the output for SummarizeChatRelationshipV1 prompt
type SummarizeChatRelationshipV1Output struct {
	RelationshipSummary string `json:"relationship_summary"`
}

// SummarizeChatRelationshipV2Output represents the output for SummarizeChatRelationshipV2 prompt
type SummarizeChatRelationshipV2Output struct {
	RelationshipSummary string `json:"relationship_summary"`
}

// SummarizeConversationV2Output represents the output for SummarizeConversationV2 prompt
type SummarizeConversationV2Output struct {
	Summary string `json:"summary"`
}

// SummarizeDayV1Output represents the output for SummarizeDayV1 prompt
type SummarizeDayV1Output []string

// SummarizeIdeasV1Output represents the output for SummarizeIdeasV1 prompt
type SummarizeIdeasV1Output struct {
	Summary string `json:"summary"`
}

// TaskDecompV3Output represents the output for TaskDecompV3 prompt
type TaskDecompV3Output struct {
	Schedule []struct {
		Task            string `json:"task"`
		DurationMinutes int    `json:"duration_in_minutes"`
		MinutesLeft     int    `json:"minutes_left"`
	} `json:"schedule"`
}

// WakeUpHourV2Output represents the output for WakeUpHourV2 prompt
type WakeUpHourV2Output struct {
	WakeUpTime string `json:"wake_up_time"`
}

// WhisperInnerThoughtV1Output represents the output for WhisperInnerThoughtV1 prompt
type WhisperInnerThoughtV1Output struct {
	Statement string `json:"statement"`
}

type DescribeAgentFeelingsV1Output struct {
	Feelings string `json:"feelings"`
}

type ExtractSchedulingInformationV1Output struct {
	Memory string `json:"memory"`
}

type GenerateCurrentlyV1Output struct {
	Status string `json:"status"`
}

type ReviseDailyRequirementsV1Output struct {
	Day string `json:"day"`
}

// ValenceChatV1Output represents the output for ValenceChatV1 prompt
type ValenceChatV1Output struct {
	Reasoning string `json:"reasoning"`
	Valence   int    `json:"valence"`
}

// ValenceEventV1Output represents the output for ValenceEventV1 prompt
type ValenceEventV1Output struct {
	Reasoning string `json:"reasoning"`
	Valence   int    `json:"valence"`
}

// ValenceThoughtV1Output represents the output for ValenceThoughtV1 prompt
type ValenceThoughtV1Output struct {
	Reasoning string `json:"reasoning"`
	Valence   int    `json:"valence"`
}
