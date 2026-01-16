package simulationloader

import (
	"encoding/json"
	"strings"
	"time"
)

type MazeMetaInfo struct {
	WorldName          string `json:"world_name"`
	MazeWidth          int    `json:"maze_width"`
	MazeHeight         int    `json:"maze_height"`
	SquareTileSize     int    `json:"sq_tile_size"`
	SpecialConstraints string `json:"special_constraints"`
}

type (
	StartDate   time.Time
	CurrentTime time.Time
	MemoryTime  time.Time
)

const (
	StartDateFormat   = "January 02, 2006"
	CurrentTimeFormat = "January 02, 2006, 15:04:05"
	MemoryTimeFormat  = "2006-01-02 15:04:05"
)

func (t StartDate) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)

	// Format as quoted string
	s := tt.Format(StartDateFormat)
	return json.Marshal(s)
}

func (t *StartDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)

	d, err := time.Parse(StartDateFormat, s)
	if err != nil {
		return err
	}

	*t = StartDate(d)
	return nil
}

func (t CurrentTime) MarshalJSON() ([]byte, error) {
	// Mirror UnmarshalJSON behavior
	tt := time.Time(t)

	// Zero value → null
	if tt.IsZero() {
		return []byte("null"), nil
	}

	// Format as quoted string
	s := tt.Format(CurrentTimeFormat)
	return json.Marshal(s)
}

func (t *CurrentTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*t = CurrentTime(time.Time{})
		return nil
	}

	s := strings.Trim(string(b), `"`)

	d, err := time.Parse(CurrentTimeFormat, s)
	if err != nil {
		return err
	}

	*t = CurrentTime(d)
	return nil
}

func (t MemoryTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)

	// Format as quoted string
	s := tt.Format(MemoryTimeFormat)
	return json.Marshal(s)
}

func (t *MemoryTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)

	d, err := time.Parse(MemoryTimeFormat, s)
	if err != nil {
		return err
	}

	*t = MemoryTime(d)
	return nil
}

type SimulationMeta struct {
	ForkSimCode    string      `json:"fork_sim_code"`
	StartDate      StartDate   `json:"start_date"`
	CurrTime       CurrentTime `json:"curr_time"`
	SecondsPerStep int         `json:"sec_per_step"`
	MazeName       string      `json:"maze_name"`
	PersonaNames   []string    `json:"persona_names"`
	Step           int         `json:"step"`
	BackupInterval int         `json:"backup_interval"`
}

type Persona struct{}

type EnvironmentPersona struct {
	Maze string `json:"maze"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}
type Environment struct {
	Personas map[string]EnvironmentPersona
}

func (e Environment) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Personas)
}

func (e *Environment) UnmarshalJSON(b []byte) error {
	personas := map[string]EnvironmentPersona{}

	if err := json.Unmarshal(b, &personas); err != nil {
		return err
	}

	e.Personas = personas
	return nil
}

type KwStength struct {
	Thoughts map[string]int `json:"kw_strength_thought"`
	Events   map[string]int `json:"kw_strength_event"`
}

type MemoryNode struct {
	NodeCount    int         `json:"node_count"`
	TypeCount    int         `json:"type_count"`
	Type         string      `json:"type"`
	Depth        int         `json:"depth"`
	Created      MemoryTime  `json:"created"`
	Expiration   *MemoryTime `json:"expiration"`
	Subject      string      `json:"subject"`
	Predicate    string      `json:"predicate"`
	Object       string      `json:"object"`
	Description  string      `json:"description"`
	EmbeddingKey string      `json:"embedding_key"`
	Poignancy    int         `json:"poignancy"`
	Valence      int         `json:"valence"`
	Keywords     []string    `json:"keywords"`
	Filling      interface{} `json:"filling"`
}

type PersonaState struct {
	VisionR                 int            `json:"vision_r"`
	AttBandwidth            int            `json:"att_bandwidth"`
	Retention               int            `json:"retention"`
	CurrTime                CurrentTime    `json:"curr_time"`
	CurrTile                []int          `json:"curr_tile"`
	DailyPlanReq            string         `json:"daily_plan_req"`
	Name                    string         `json:"name"`
	FirstName               string         `json:"first_name"`
	LastName                string         `json:"last_name"`
	Age                     int            `json:"age"`
	Innate                  string         `json:"innate"`
	Learned                 string         `json:"learned"`
	Currently               string         `json:"currently"`
	Lifestyle               string         `json:"lifestyle"`
	LivingArea              string         `json:"living_area"`
	ConceptForget           int            `json:"concept_forget"`
	DailyReflectionTime     int            `json:"daily_reflection_time"`
	DailyReflectionSize     int            `json:"daily_reflection_size"`
	OverlapReflectTh        int            `json:"overlap_reflect_th"`
	KwStrgEventReflectTh    int            `json:"kw_strg_event_reflect_th"`
	KwStrgThoughtReflectTh  int            `json:"kw_strg_thought_reflect_th"`
	RecencyW                float64        `json:"recency_w"`
	RelevanceW              float64        `json:"relevance_w"`
	ImportanceW             float64        `json:"importance_w"`
	ValenceW                float64        `json:"valence_w"`
	RecencyDecay            float64        `json:"recency_decay"`
	ImportanceTriggerMax    int            `json:"importance_trigger_max"`
	ImportanceTriggerCurr   int            `json:"importance_trigger_curr"`
	ImportanceEleN          int            `json:"importance_ele_n"`
	ThoughtCount            int            `json:"thought_count"`
	DailyReq                []string       `json:"daily_req"`
	FDailySchedule          []Plan         `json:"f_daily_schedule"`
	FDailyScheduleHourlyOrg []Plan         `json:"f_daily_schedule_hourly_org"`
	ActAddress              string         `json:"act_address"`
	ActStartTime            CurrentTime    `json:"act_start_time"`
	ActDuration             int            `json:"act_duration"`
	ActDescription          string         `json:"act_description"`
	ActPronunciatio         string         `json:"act_pronunciatio"`
	ActEvent                SPO            `json:"act_event"`
	ActObjDescription       string         `json:"act_obj_description"`
	ActObjPronunciatio      string         `json:"act_obj_pronunciatio"`
	ActObjEvent             SPO            `json:"act_obj_event"`
	ChattingWith            *string        `json:"chatting_with"`
	Chat                    []Utterance    `json:"chat"`
	ChattingWithBuffer      map[string]int `json:"chatting_with_buffer"`
	ChattingEndTime         *CurrentTime   `json:"chatting_end_time"`
	ActPathSet              bool           `json:"act_path_set"`
	PlannedPath             []Position     `json:"planned_path"`
}

type Plan struct {
	Activity string
	Duration int
}

func (p Plan) MarshalJSON() ([]byte, error) {
	tmp := [2]any{p.Activity, p.Duration}
	return json.Marshal(tmp)
}

func (p *Plan) UnmarshalJSON(data []byte) error {
	// Expect exactly: [string, number]
	var tmp [2]json.RawMessage
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[0], &p.Activity); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[1], &p.Duration); err != nil {
		return err
	}

	return nil
}

type Position struct {
	X, Y int
}

func (pos Position) MarshalJSON() ([]byte, error) {
	tmp := [2]interface{}{pos.X, pos.Y}
	return json.Marshal(tmp)
}

func (p *Position) UnmarshalJSON(data []byte) error {
	// Expect exactly: [number, number]
	var tmp [2]json.RawMessage
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[0], &p.X); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[1], &p.Y); err != nil {
		return err
	}

	return nil
}

type SPO struct {
	Subject, Predicate, Object string
}

func (spo SPO) MarshalJSON() ([]byte, error) {
	tmp := [3]interface{}{spo.Subject, spo.Predicate, spo.Object}
	return json.Marshal(tmp)
}

func (spo *SPO) UnmarshalJSON(data []byte) error {
	var tmp [3]json.RawMessage
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[0], &spo.Subject); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[1], &spo.Predicate); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[2], &spo.Object); err != nil {
		return err
	}

	return nil
}

type Utterance struct {
	Speaker, Utterance string
}

func (u Utterance) MarshalJSON() ([]byte, error) {
	// Encode as: [speaker, utterance]
	tmp := [2]interface{}{u.Speaker, u.Utterance}
	return json.Marshal(tmp)
}

func (u *Utterance) UnmarshalJSON(data []byte) error {
	// Expect exactly: [string, number]
	var tmp [2]json.RawMessage
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[0], &u.Speaker); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[1], &u.Utterance); err != nil {
		return err
	}

	return nil
}
