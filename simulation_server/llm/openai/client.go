package openai

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/llm"
	"github.com/fvdveen/generative_agents/simulation_server/memory"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

const (
	hourFormat     = "3:04pm"
	hourFormat24   = "15:04"
	dateHourFormat = "January 02, 2006, 15:04:05"
)

//go:embed v5/*
var promptFiles embed.FS

type schema struct {
	Name   string
	Schema map[string]any
}

type prompt struct {
	name     string
	schema   schema
	template *template.Template
}

var templateFuncs template.FuncMap = template.FuncMap{
	"add1": func(i int) int {
		return i + 1
	},
	"PathLevelSector": func() memory.PathLevel { return memory.PathLevelSector },
	"PathLevelArena":  func() memory.PathLevel { return memory.PathLevelArena },
	"join":            strings.Join,
}

func loadPrompts() map[string]prompt {
	prompts := map[string]prompt{}

	dirs, err := promptFiles.ReadDir("v5")
	if err != nil {
		panic(fmt.Sprintf("Could not read prompt directory: %v", err))
	}

	for _, dir := range dirs {
		name := dir.Name()
		if name == "." || name == ".." || !dir.IsDir() {
			continue
		}

		content, err := promptFiles.ReadFile(fmt.Sprintf("v5/%s/schema.json", name))
		if err != nil {
			panic(fmt.Sprintf("Could not read schema file for %s: %v", name, err))
		}

		schema := schema{Name: name, Schema: map[string]any{}}
		if err = json.Unmarshal(content, &schema.Schema); err != nil {
			panic(fmt.Sprintf("Could not unmarschal schema for %s: %v", name, err))
		}

		content, err = promptFiles.ReadFile(fmt.Sprintf("v5/%s/prompt.txt", name))
		if err != nil {
			panic(fmt.Sprintf("Could not read template file for %s: %v", name, err))
		}

		template := template.Must(template.
			New(name).
			Funcs(templateFuncs).
			Option("missingkey=error").
			Parse(string(content)))

		prompts[name] = prompt{name, schema, template}
	}

	return prompts
}

var prompts = loadPrompts()

type ClientOpt func(c *Client)

func WithAPIKey(key string) ClientOpt {
	return func(c *Client) {
		c.apiKey = key
	}
}

func WithURL(url string) ClientOpt {
	return func(c *Client) {
		c.url = url
	}
}

func WithLogger(logger *slog.Logger) ClientOpt {
	return func(c *Client) {
		c.logger = logger
	}
}

func WithTextModel(model string) ClientOpt {
	return func(c *Client) {
		c.textModel = model
	}
}

func WithEmbeddingsModel(model string) ClientOpt {
	return func(c *Client) {
		c.embeddingModel = model
	}
}

type Client struct {
	client openai.Client
	logger *slog.Logger

	apiKey string
	url    string

	textModel      string
	embeddingModel string
	maxRetries     int

	llmSeq atomic.Uint64
}

func New(opts ...ClientOpt) *Client {
	client := &Client{textModel: "gpt-5-nano", embeddingModel: "text-embedding-ada-002", maxRetries: 5, logger: slog.Default()}

	for _, opt := range opts {
		opt(client)
	}

	openaiOpts := []option.RequestOption{option.WithAPIKey(client.apiKey)}
	if client.url != "" {
		openaiOpts = append(openaiOpts, option.WithBaseURL(client.url))
	}

	client.client = openai.NewClient(openaiOpts...)

	return client
}

func (c *Client) newID() string {
	n := c.llmSeq.Add(1)
	return fmt.Sprintf("llm-%d", n)
}

func (c *Client) responseParams(prompt string, schema schema) responses.ResponseNewParams {
	r := responses.ResponseNewParams{
		Model:     c.textModel,
		Reasoning: shared.ReasoningParam{Effort: "low"},
		Input: responses.ResponseNewParamsInputUnion{
			OfString: param.NewOpt(prompt),
		},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigParamOfJSONSchema(schema.Name, schema.Schema),
		},
	}

	return r
}

func (c *Client) doRequest(ctx context.Context, promptText string, schema schema, output any) (*responses.Response, error) {
	resp, err := c.client.Responses.New(ctx, c.responseParams(promptText, schema))
	if err != nil {
		return nil, fmt.Errorf("could not execute prompt: %w", err)
	}

	raw := resp.OutputText()

	if err := json.Unmarshal([]byte(raw), output); err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %w", err)
	}

	return resp, nil
}

func isJSONUnmarshalError(err error) bool {
	if err == nil {
		return false
	}

	var (
		syntaxErr  *json.SyntaxError
		typeErr    *json.UnmarshalTypeError
		invalidErr *json.InvalidUnmarshalError
	)

	return errors.As(err, &syntaxErr) ||
		errors.As(err, &typeErr) ||
		errors.As(err, &invalidErr)
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	// 8–12 bytes is plenty for logging fingerprints
	return hex.EncodeToString(sum[:8]) // 16 hex chars
}

// doRequestWithRetry calls doRequest with retry logic for JSON unmarshalling or validation failures
func (c *Client) doRequestWithRetry(ctx context.Context, prompt prompt, params any, output any, validationFn func() error) error {
	var lastErr error

	var wr strings.Builder
	if err := prompt.template.Execute(&wr, params); err != nil {
		return fmt.Errorf("could not execute prompt template: %w", err)
	}

	promptText := wr.String()

	llmID := c.newID()
	log := c.logger.With(
		slog.String("llm_id", llmID),
		slog.String("prompt_name", prompt.name),
		slog.Int("max_retries", c.maxRetries),
		slog.String("type", "llm_call"),
	)

	log.Info("llm_call_start",
		slog.String("type", "llm_call"),
		slog.String("phase", "start"),
		slog.String("prompt_hash", hashString(promptText)),
		slog.Int("prompt_length", len(promptText)),
	)

	start := time.Now()
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		resp, err := c.doRequest(ctx, promptText, prompt.schema, output)
		if err != nil {
			lastErr = err
			// Only retry on JSON unmarshalling errors
			if isJSONUnmarshalError(err) {
				log.Warn("llm_retry",
					slog.String("phase", "retry"),
					slog.Int("attempt", attempt+1),
					slog.String("reason", "json_unmarshal"),
					slog.Any("err", err),
					slog.String("response_hash", hashString(resp.OutputText())),
					slog.Int("response_len", len(resp.OutputText())),
				)
				continue
			}

			log.Error("llm_call_fail",
				"type", "llm_call",
				"phase", "fail",
				"attempt", attempt+1,
				"total_latency", time.Since(start),
				"err", err,
			)
			return err
		}

		// If validation function is provided, run it and retry on failure
		if validationFn != nil {
			if err := validationFn(); err != nil {
				lastErr = err
				log.Warn("llm_retry",
					"type", "llm_call",
					"phase", "retry",
					"attempt", attempt+1,
					"reason", "validation",
					"err", err,
				)
				continue
			}
		}

		log.Info("llm_call_ok",
			"type", "llm_call",
			"phase", "ok",
			"attempts_total", attempt+1,
			"total_latency", time.Since(start),
		)
		// Success
		return nil
	}

	log.Error("llm_call_fail",
		"type", "llm_call",
		"phase", "fail",
		"attempts_total", c.maxRetries,
		"total_latency", time.Since(start),
		"prompt_raw", promptText,
		"err", lastErr,
	)

	return fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

func (c *Client) GenerateEmbedding(str string) []float64 {
	str = strings.Replace(str, "\n", " ", -1)
	res, err := c.client.Embeddings.New(context.Background(), openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.NewOpt(str),
		},
		Model:          c.embeddingModel,
		EncodingFormat: "float",
	})
	if err != nil {
		panic(fmt.Sprintf("Could not generate embeddings for %s: %v", str, err))
	}

	return res.Data[0].Embedding
}

// Generates an importance score for a memory of a specific type based off of the
// persona's personality and the event description.
func (c *Client) GenerateImportanceScore(p llm.Persona, nt memory.NodeType, description string) int {
	switch nt {
	case memory.NodeTypeChat:
		fmt.Println("chat importance scores should be generated by GenerateImportanceScoreChat not GenerateImportanceScore")
		return c.GenerateImportanceScoreChat(p, p.CurrentChat(), description)
	case memory.NodeTypeEvent:
		return c.generateImportanceEvent(p, description)
	case memory.NodeTypeThought:
		return c.generateImportanceThought(p, description)
	default:
		panic(fmt.Sprintf("unexpected memory.NodeType: %#v", nt))
	}
}

func (c *Client) generateImportanceThought(p llm.Persona, thought string) int {
	prompt := prompts["poignancy_thought_v1"]

	in := GeneratePoignancyThoughtV1Input{
		Persona: p,
		Thought: thought,
	}

	var out PoignancyThoughtV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Poignancy
}

func (c *Client) generateImportanceEvent(p llm.Persona, event string) int {
	prompt := prompts["poignancy_event_v1"]

	in := GeneratePoignancyEventV1Input{
		Persona: p,
		Event:   event,
	}

	var out PoignancyEventV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Poignancy
}

func (c *Client) GenerateImportanceScoreChat(p llm.Persona, transcript []memory.Utterance, description string) int {
	prompt := prompts["poignancy_chat_v1"]

	in := GeneratePoignancyChatV1Input{
		Persona:      p,
		Description:  description,
		Conversation: transcript,
	}

	var out PoignancyChatV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Poignancy
}

// GenerateValenceScore implements llm.Cognition.
func (c *Client) GenerateValenceScore(p llm.Persona, nt memory.NodeType, description string) int {
	switch nt {
	case memory.NodeTypeChat:
		fmt.Println("chat importance scores should be generated by GenerateImportanceScoreChat not GenerateImportanceScore")
		return c.GenerateValenceScoreChat(p, p.CurrentChat(), description)
	case memory.NodeTypeEvent:
		return c.generateValenceEvent(p, description)
	case memory.NodeTypeThought:
		return c.generateValenceThought(p, description)
	default:
		panic(fmt.Sprintf("unexpected memory.NodeType: %#v", nt))
	}
}

func (c *Client) generateValenceThought(p llm.Persona, description string) int {
	prompt := prompts["valence_thought_v1"]

	in := GenerateValenceThoughtV1Input{
		Persona: p,
		Thought: description,
	}

	var out ValenceThoughtV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Valence
}

func (c *Client) generateValenceEvent(p llm.Persona, description string) int {
	prompt := prompts["valence_event_v1"]

	in := GenerateValenceEventV1Input{
		Persona: p,
		Event:   description,
	}

	var out ValenceEventV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Valence
}

func (c *Client) GenerateValenceScoreChat(p llm.Persona, transcript []memory.Utterance, description string) int {
	prompt := prompts["valence_chat_v1"]

	in := GenerateValenceChatV1Input{
		Persona:      p,
		Description:  description,
		Conversation: transcript,
	}

	var out ValenceChatV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Valence
}

// Generates the wake up hour for the next day based off of the persona's personality.
func (c *Client) GenerateWakeUpHour(p llm.Persona) time.Time {
	prompt := prompts["wake_up_hour_v2"]

	in := WakeUpHourV2Input{
		Persona: p,
	}

	var out WakeUpHourV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	time, err := time.Parse(hourFormat, strings.ToLower(strings.Replace(out.WakeUpTime, " ", "", -1)))
	if err != nil {
		panic(fmt.Sprintf("could not parse output time: %v", err))
	}

	return time
}

// Generates the first daily plan for a persona.
func (c *Client) GenerateDailyPlan(p llm.Persona, wakeUpHour time.Time) []string {
	prompt := prompts["daily_planning_v7"]

	in := DailyPlanningV7Input{
		Persona:     p,
		WakeUpHour:  wakeUpHour.Format(hourFormat),
		CurrentDate: p.CurrentTime().Format("Monday January 2"),
	}

	var out DailyPlanningV7Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Schedule
}

// Generates an hour schedule for a new day.
func (c *Client) GenerateHourlySchedule(p llm.Persona, wakeUpHour time.Time) []llm.Plan {
	prompt := prompts["generate_hourly_schedule_v2"]

	in := GenerateHourlyScheduleV2Input{
		Persona: p,
	}

	var out GenerateHourlyScheduleV2Output

	// Validation function to check schedule constraints
	validationFn := func() error {
		// These are just some sanity checks to make sure the model aligned with the prompt
		if len(out.Schedule) != 24 {
			return fmt.Errorf("generated schedule does not have 24 items")
		}

		for i, t := range []string{
			"12:00am", "01:00am", "02:00am", "03:00am", "04:00am",
			"05:00am", "06:00am", "07:00am", "08:00am", "09:00am",
			"10:00am", "11:00am", "12:00pm", "01:00pm", "02:00pm",
			"03:00pm", "04:00pm", "05:00pm", "06:00pm", "07:00pm",
			"08:00pm", "09:00pm", "10:00pm", "11:00pm",
		} {
			got := strings.ToLower(strings.Replace(out.Schedule[i].Time, " ", "", -1))
			if got != t && got != strings.TrimPrefix(t, "0") {
				return fmt.Errorf("generated schedule has wrong time, expected: %s, got: %s", t, out.Schedule[i].Time)
			}
		}
		return nil
	}

	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, validationFn); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	plans := []llm.Plan{}
	for _, p := range out.Schedule {
		dur := 60
		if len(plans) > 0 && plans[len(plans)-1].Activity == p.Activity {
			plans[len(plans)-1].Duration += dur
		} else {
			plans = append(plans, llm.Plan{Duration: dur, Activity: p.Activity})
		}
	}

	return plans
}

// Generates a list of sub-plans that the given plan should consist of
func (c *Client) GeneratePlanDecomposition(p llm.Persona, plan llm.Plan) []llm.Plan {
	prompt := prompts["task_decomp_v3"]

	in := TaskDecompV3Input{
		Persona:          p,
		CurrentDate:      p.CurrentTime().Format("January 02, 2006"),
		Activity:         plan.Activity,
		ActivityDuration: plan.Duration,
	}

	hourlySchedule := p.OriginalHourlySchedule()
	hourlyScheduleIdx := p.OriginalHourlyScheduleIndex()

	indices := []int{hourlyScheduleIdx}
	if len(hourlySchedule) >= hourlyScheduleIdx+1 {
		indices = append(indices, hourlyScheduleIdx+1)
	}
	if len(hourlySchedule) >= hourlyScheduleIdx+2 {
		indices = append(indices, hourlyScheduleIdx+2)
	}

	for _, idx := range indices {
		if idx >= len(hourlySchedule) {
			continue
		}

		startMin := 0
		for i := 0; i < idx; i += 1 {
			startMin += hourlySchedule[i].Duration
		}
		endMin := startMin + hourlySchedule[idx].Duration
		start := p.StartOfDay().
			Add(time.Minute * time.Duration(startMin))
		end := p.StartOfDay().
			Add(time.Minute * time.Duration(endMin))
		in.Schedule = append(in.Schedule, struct {
			Activity  string
			StartTime string
			EndTime   string
		}{
			StartTime: start.Format("15:04PM"),
			EndTime:   end.Format("15:04PM"),
			Activity:  hourlySchedule[idx].Activity,
		})

		if idx == hourlyScheduleIdx+1 {
			in.ActivityStartTime = start.Format("15:04PM")
			in.ActivityEndTime = end.Format("15:04PM")
		}
	}

	var out TaskDecompV3Output

	// Validation function to check task decomposition constraints
	validationFn := func() error {
		totalTime := plan.Duration
		for _, task := range out.Schedule {
			totalTime -= task.DurationMinutes
			if totalTime != task.MinutesLeft {
				return fmt.Errorf("task time left does not match, expected: %d, got: %d", totalTime, task.MinutesLeft)
			}
		}

		if totalTime != 0 {
			return fmt.Errorf("tasks do not add up to expected time, expected: %d, got: %d", plan.Duration, plan.Duration-totalTime)
		}
		return nil
	}

	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, validationFn); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	tasks := []llm.Plan{}
	for _, task := range out.Schedule {
		tasks = append(tasks, llm.Plan{
			Activity: task.Task,
			Duration: task.DurationMinutes,
		})
	}

	return tasks
}

// Generates an updated schedule in response to an event
func (c *Client) GenerateReactionScheduleUpdate(p llm.Persona, inserted llm.Plan, startTime, endTime time.Time) []llm.Plan {
	prompt := prompts["new_decomp_schedule_v2"]

	originalPlans := []struct {
		StartTime string
		EndTime   string
		Activity  string
	}{}

	truncatedPlans := []struct {
		StartTime string
		EndTime   string
		Activity  string
	}{}

	generatedPlans := []llm.Plan{}

	durationSum := p.StartOfDay()
	planningDuration := int(endTime.Sub(startTime).Minutes())
	for _, plan := range p.DailySchedule() {
		end := durationSum.Add(time.Duration(plan.Duration) * time.Minute)
		if !durationSum.Before(startTime) && !end.After(endTime) {
			originalPlans = append(originalPlans, struct {
				StartTime string
				EndTime   string
				Activity  string
			}{
				StartTime: durationSum.Format(hourFormat24),
				EndTime:   end.Format(hourFormat24),
				Activity:  plan.Activity,
			})

			if !end.After(p.CurrentTime()) {
				truncatedPlans = append(truncatedPlans, struct {
					StartTime string
					EndTime   string
					Activity  string
				}{
					StartTime: durationSum.Format(hourFormat24),
					EndTime:   end.Format(hourFormat24),
					Activity:  plan.Activity,
				})
				generatedPlans = append(generatedPlans, plan)
				planningDuration -= plan.Duration
			} else if durationSum.Before(p.CurrentTime()) && end.After(p.CurrentTime()) {
				trunc := end.Sub(p.CurrentTime())
				truncDur := plan.Duration - int(math.Ceil(trunc.Minutes()))
				if truncDur != 0 {
					truncatedPlans = append(truncatedPlans, struct {
						StartTime string
						EndTime   string
						Activity  string
					}{
						StartTime: durationSum.Format(hourFormat24),
						EndTime:   p.CurrentTime().Format(hourFormat24),
						Activity:  plan.Activity,
					})

					generatedPlans = append(generatedPlans, llm.Plan{Activity: plan.Activity, Duration: truncDur})
					planningDuration -= truncDur
				}
			}

		}

		durationSum = durationSum.Add(time.Duration(plan.Duration) * time.Minute)
	}

	// NOTE(Friso): At this point im not really sure if its a good idea to let the LLM itself decide how long the interruption is
	// or if I should enfore the duration, right now ill let the LLM decide as thats how it happens in the original paper.

	// truncatedPlans = append(truncatedPlans, struct {
	// 	StartTime string
	// 	EndTime   string
	// 	Activity  string
	// }{
	// 	StartTime: p.CurrentTime().Format(hourFormat24),
	// 	EndTime:   p.CurrentTime().Add(time.Duration(inserted.Duration) * time.Minute).Format(hourFormat24),
	// 	Activity:  inserted.Activity,
	// })
	// generatedPlans = append(generatedPlans, inserted)
	// planningDuration -= inserted.Duration

	in := NewDecompScheduleV2Input{
		Persona:           p,
		OriginalStartTime: startTime.Format(hourFormat24),
		OriginalEndTime:   endTime.Format(hourFormat24),
		PlanningFromTime:  endTime.Add(-time.Duration(planningDuration) * time.Minute).Format(hourFormat24),
		OriginalPlans:     originalPlans,
		TruncatedPlans:    truncatedPlans,
		Inserted:          inserted,
	}

	var out NewDecompScheduleV2Output

	// Validation function to check reaction schedule constraints
	validationFn := func() error {
		outDurSum := 0
		for _, plan := range out.Schedule {
			outDurSum += plan.DurationMinutes
		}

		if outDurSum != planningDuration {
			return fmt.Errorf("Unexpected duration of reaction decomposistion planning, expected: %d, got: %d", planningDuration, outDurSum)
		}

		totalDur := 0
		for _, plan := range out.Schedule {
			totalDur += plan.DurationMinutes
		}
		for _, plan := range generatedPlans {
			totalDur += plan.Duration
		}

		if totalDur != int(endTime.Sub(startTime).Minutes()) {
			return fmt.Errorf("Unexpected duration of full reaction decomposistion planning, expected: %d, got: %d", int(endTime.Sub(startTime).Minutes()), totalDur)
		}
		return nil
	}

	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, validationFn); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	for _, plan := range out.Schedule {
		generatedPlans = append(generatedPlans, llm.Plan{
			Activity: plan.Action,
			Duration: plan.DurationMinutes,
		})
	}

	return generatedPlans
}

var actionRe = regexp.MustCompile(`^(.*) \((.*)\)$`)

// Generates the sector an activity should take place in
func (c *Client) GenerateActivitySector(p llm.Persona, maze llm.Maze, activity string, world string) string {
	prompt := prompts["action_location_sector_v3"]

	action, subAction := activity, activity
	if strings.Contains(activity, "(") {
		m := actionRe.FindStringSubmatch(activity)
		if m != nil {
			action = m[1]
			subAction = m[2]
		}
	}

	path := maze.GetTile(p.Position()).Path
	in := ActionLocationSectorV3Input{
		Persona:         p,
		CurrentLocation: path,
		Action:          action,
		SubAction:       subAction,
	}

	var out ActionLocationSectorV3Output

	// Validation function to check if path exists
	validationFn := func() error {
		new := path.AtLevel(memory.PathLevelSector).Copy(memory.PathWithSector(out.Output))
		if !maze.Exists(new) {
			return fmt.Errorf("path does not exist: %s", new.ToString())
		}
		return nil
	}

	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, validationFn); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Output
}

// Generates the arena an activity should take place in
func (c *Client) GenerateActivityArena(p llm.Persona, maze llm.Maze, activity string, world string, sector string) string {
	prompt := prompts["action_location_arena_v1"]

	action, subAction := activity, activity
	if strings.Contains(activity, "(") {
		m := actionRe.FindStringSubmatch(activity)
		if m != nil {
			action = m[1]
			subAction = m[2]
		}
	}

	path := maze.GetTile(p.Position()).Path
	in := ActionLocationArenaV1Input{
		Persona:         p,
		CurrentLocation: path,
		TargetLocation: memory.NewPath(
			memory.PathWithWorld(world),
			memory.PathWithSector(sector)),
		Activity:          action,
		SubAction:         subAction,
		DestinationSector: sector,
	}

	var out ActionLocationSectorV3Output

	// Validation function to check if path exists
	validationFn := func() error {
		new := path.AtLevel(memory.PathLevelArena).Copy(memory.PathWithSector(sector), memory.PathWithArena(out.Output))
		if !maze.Exists(new) {
			return fmt.Errorf("path does not exist: %s", new.ToString())
		}
		return nil
	}

	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, validationFn); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Output
}

// Generates the object that should be used for an activity
func (c *Client) GenerateActivityObject(p llm.Persona, maze llm.Maze, activity string, path memory.Path) string {
	prompt := prompts["action_object_v3"]

	in := ActionObjectV1Input{
		Persona:        p,
		TargetLocation: path,
		Activity:       activity,
	}

	var out ActionObjectV1Output

	// Validation function to check if path exists
	validationFn := func() error {
		new := path.AtLevel(memory.PathLevelObject).Copy(memory.PathWithObject(out.Output))
		if !maze.Exists(new) {
			return fmt.Errorf("path does not exist: %s", new.ToString())
		}
		return nil
	}

	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, validationFn); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Output
}

// Generates a pronunciato (2 emojis) representing the current activity taking place
func (c *Client) GenerateActivityPronunciato(p llm.Persona, activity string) string {
	prompt := prompts["generate_pronunciatio_v2"]

	in := GeneratePronunciatioV2Input{
		Activity: activity,
	}

	var out GeneratePronunciatioV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Emoji
}

// Generates a SPO (activity subject-predicate-object) triple
func (c *Client) GenerateActivitySPO(p llm.Persona, activity string) memory.SPO {
	prompt := prompts["generate_event_triple_v2"]

	in := GenerateEventTripleV2Input{
		Name:     p.Name(),
		Activity: activity,
	}

	var out GenerateEventTripleV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return memory.SPO{
		Subject:   out.Subject,
		Predicate: out.Predicate,
		Object:    out.Object,
	}
}

// Generates a description for the object that is used in the current activity
func (c *Client) GenerateActivityObjectDescription(p llm.Persona, object string, activity string) string {
	prompt := prompts["generate_obj_event_v2"]

	in := GenerateObjEventV2Input{
		Persona:  p,
		Activity: activity,
		Object:   object,
	}

	var out GenerateObjEventV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.State
}

// Generates a pronunciato (2 emojis) representing t for the object that is used in the current activity
func (c *Client) GenerateActivityObjectPronunciato(p llm.Persona, activityObjectDescription string) string {
	prompt := prompts["generate_pronunciatio_v2"]

	in := GeneratePronunciatioV2Input{
		Activity: activityObjectDescription,
	}

	var out GeneratePronunciatioV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Emoji
}

// Generates a SPO (activity subject-predicate-object) triple
func (c *Client) GenerateActivityObjectSPO(p llm.Persona, object string, activityObjectDescription string) memory.SPO {
	prompt := prompts["generate_event_triple_v2"]

	in := GenerateEventTripleV2Input{
		Name:     object,
		Activity: activityObjectDescription,
	}

	var out GenerateEventTripleV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return memory.SPO{
		Subject:   out.Subject,
		Predicate: out.Predicate,
		Object:    out.Object,
	}
}

// Generates whether Persona init wants to talk to persona target
func (c *Client) GenerateDecideToTalk(init, target llm.Persona, events, thoughts []memory.NodeId) bool {
	prompt := prompts["decide_to_talk_v3"]

	var ctx strings.Builder
	if len(events) != 0 {
		ctx.WriteString("Observations: ")
		for _, node := range events {
			event := init.GetMemory(node)
			desc := strings.Replace(event.Description, "is", "was", 1)
			ctx.WriteString(desc + ". ")
		}
	}
	if len(thoughts) != 0 {
		ctx.WriteString(", Thoughts: ")
		for _, node := range thoughts {
			thought := init.GetMemory(node)
			ctx.WriteString(thought.Description + ". ")
		}
	}
	if len(thoughts) == 0 && len(events) == 0 {
		ctx.WriteString("None")
	}

	var lastChatTime, lastChatTopic string
	chatId, ok := init.LastChat(target.Name())
	if ok {
		node := init.GetMemory(chatId)
		lastChatTopic = node.Description
		lastChatTime = node.Created.Format(dateHourFormat)
	}

	initStat, targetStat := init.ActivityDescription(), target.ActivityDescription()
	initAct := actionRe.FindStringSubmatch(initStat)
	if initAct != nil {
		initStat = initAct[2]
	}
	targetAct := actionRe.FindStringSubmatch(targetStat)
	if targetAct != nil {
		targetStat = targetAct[2]
	}

	if (len(init.PlannedPath()) == 0) && !strings.Contains(init.ActivityDescription(), "waiting") {
		initStat = fmt.Sprintf("%s is already %s", init.Name(), initStat)
	} else if strings.Contains(init.ActivityDescription(), "waiting") {
		initStat = fmt.Sprintf("%s is %s", init.Name(), initStat)
	} else {
		initStat = fmt.Sprintf("%s in on the way to %s", init.Name(), initStat)
	}

	if (len(target.PlannedPath()) == 0) && !strings.Contains(target.ActivityDescription(), "waiting") {
		targetStat = fmt.Sprintf("%s is already %s", target.Name(), targetStat)
	} else if strings.Contains(target.ActivityDescription(), "waiting") {
		targetStat = fmt.Sprintf("%s is %s", target.Name(), targetStat)
	} else {
		targetStat = fmt.Sprintf("%s in on the way to %s", target.Name(), targetStat)
	}

	in := DecideToTalkV3Input{
		Initiator:       init,
		Target:          target,
		InitiatorStatus: initStat,
		TargetStatus:    targetStat,
		Context:         ctx.String(),
		CurrentTime:     init.CurrentTime().Format(hourFormat24),
		LastChatTime:    lastChatTime,
		LastChatTopic:   lastChatTopic,
	}

	var out DecideToTalkV3Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return strings.ToLower(out.ShouldTalk) == "yes"
}

// Generates whether init should wait until target has finished their activity before approaching them,
// or init should continue with their own activity.
// NOTE(Friso): In the original code this is called generate_decide_to_react, but this name is more apt.
func (c *Client) GenerateDecideToWait(init, target llm.Persona, events, thoughts []memory.NodeId) (wait bool) {
	prompt := prompts["decide_to_react_v2"]

	var ctx strings.Builder
	if len(events) != 0 {
		ctx.WriteString("Observations: ")
		for _, node := range events {
			event := init.GetMemory(node)
			desc := strings.Replace(event.Description, "is", "was", 1)
			ctx.WriteString(desc + ". ")
		}
	}
	if len(thoughts) != 0 {
		ctx.WriteString(", Thoughts: ")
		for _, node := range thoughts {
			thought := init.GetMemory(node)
			ctx.WriteString(thought.Description + ". ")
		}
	}
	if len(thoughts) == 0 && len(events) == 0 {
		ctx.WriteString("None")
	}

	initStat, targetStat := init.ActivityDescription(), target.ActivityDescription()
	initAct := actionRe.FindStringSubmatch(initStat)
	if initAct != nil {
		initStat = initAct[2]
	}
	targetAct := actionRe.FindStringSubmatch(targetStat)
	if targetAct != nil {
		targetStat = targetAct[2]
	}

	if (len(init.PlannedPath()) == 0) && !strings.Contains(init.ActivityDescription(), "waiting") {
		initStat = fmt.Sprintf("%s is already %s", init.Name(), initStat)
	} else if strings.Contains(init.ActivityDescription(), "waiting") {
		initStat = fmt.Sprintf("%s is %s", init.Name(), initStat)
	} else {
		initStat = fmt.Sprintf("%s in on the way to %s", init.Name(), initStat)
	}

	if (len(target.PlannedPath()) == 0) && !strings.Contains(target.ActivityDescription(), "waiting") {
		targetStat = fmt.Sprintf("%s is already %s", target.Name(), targetStat)
	} else if strings.Contains(target.ActivityDescription(), "waiting") {
		targetStat = fmt.Sprintf("%s is %s", target.Name(), targetStat)
	} else {
		targetStat = fmt.Sprintf("%s in on the way to %s", target.Name(), targetStat)
	}

	in := DecideToReactV2Input{
		Initiator:       init,
		Target:          target,
		InitiatorStatus: initStat,
		TargetStatus:    targetStat,
		Context:         ctx.String(),
		CurrentTime:     init.CurrentTime().Format(hourFormat24),
		TargetEndTime:   target.ActivityEndTime(target.DailyScheduleIdx()).Format(hourFormat24),
	}

	var out DecideToReactV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Choice == 1
}

func (c *Client) GenerateOneUtterance(init, target llm.Persona, maze llm.Maze, currentChat []memory.Utterance, relevant []memory.NodeId, relationship string) (utt memory.Utterance, endConversation bool) {
	prompt := prompts["iterative_convo_v2"]

	location := maze.GetTile(init.Position())

	in := IterativeConvoV2Input{
		Init:                init,
		Target:              target,
		Relevant:            relevant,
		CurrentLocation:     fmt.Sprintf("%s in %s", location.Path.Get(memory.PathLevelSector), location.Path.Get(memory.PathLevelArena)),
		RelationshipSummary: relationship,
		Conversation:        currentChat,
	}

	var out IterativeConvoV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return memory.Utterance{
		Speaker:  init.Name(),
		Sentence: out.Utterance,
	}, out.EndsConversation
}

// GenerateRelationshipSummary implements llm.Cognition.
func (c *Client) GenerateRelationshipSummary(init llm.Persona, target llm.Persona, memories []memory.NodeId) string {
	prompt := prompts["summarize_chat_relationship_v2"]

	in := SummarizeChatRelationshipV2Input{
		Init:     init,
		Target:   target,
		Memories: memories,
	}

	var out SummarizeChatRelationshipV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.RelationshipSummary
}

// Generates a summary for a conversation that a persona had
func (c *Client) GenerateConversationSummary(p llm.Persona, conversation []memory.Utterance) string {
	prompt := prompts["summarize_conversation_v2"]

	in := SummarizeConversationV2Input{
		Conversation: conversation,
	}

	var out SummarizeConversationV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Summary
}

// Generates a change in planning for p that should be remembered based off of a conversation
func (c *Client) GeneratePlanningThoughtAfterConversation(p llm.Persona, conversation []memory.Utterance) string {
	prompt := prompts["planning_thought_on_convo_v2"]

	in := PlanningThoughtOnConvoV2Input{
		Persona:      p,
		Conversation: conversation,
	}

	var out PlanningThoughtOnConvoV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.PlanningThought
}

// Generates anything noteworthy that should be remembered after a conversation
func (c *Client) GenerateMemoAfterConversation(p llm.Persona, conversation []memory.Utterance) string {
	prompt := prompts["memo_on_convo_v1"]

	in := MemoOnConvoV1Input{
		Persona:      p,
		Conversation: conversation,
	}

	var out MemoOnConvoV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Memo
}

// Generates a list of focal points to address during reflection
func (c *Client) GenerateFocalPoints(p llm.Persona, statements []memory.NodeId, numFocalPoints int) []string {
	prompt := prompts["generate_focal_pt_v2"]

	in := GenerateFocalPtV2Input{
		Persona:    p,
		Statements: statements,
		Count:      numFocalPoints,
	}

	var out GenerateFocalPtV2Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.FocalPoints
}

// Generates insights based off of the evidence presented in nodes
func (c *Client) GenerateInsightAndEvidence(p llm.Persona, nodes []memory.NodeId, insightCount int) map[string][]memory.NodeId {
	prompt := prompts["insight_and_evidence_v2"]

	in := InsightAndEvidenceV2Input{
		Persona:    p,
		Statements: nodes,
		Count:      insightCount,
	}

	var out InsightAndEvidenceV2Output

	validationFn := func() error {
		for _, i := range out.Insights {
			for _, e := range i.Reasons {
				if e-1 >= len(nodes) {
					return fmt.Errorf("insight %s has reason not in nodes, maximum: %d, got: %d", i.Insight, len(nodes), e-1)
				}
			}
		}
		return nil
	}

	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, validationFn); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	insights := map[string][]memory.NodeId{}

	for _, i := range out.Insights {
		evidence := make([]memory.NodeId, 0, len(i.Reasons))

		for _, e := range i.Reasons {
			evidence = append(evidence, nodes[e-1])
		}

		insights[i.Insight] = evidence
	}

	return insights
}

// GeneratePlanningFeelings implements llm.Cognition.
func (c *Client) GeneratePlanningFeelings(p llm.Persona, statements []string) string {
	prompt := prompts["describe_agent_feelings_v1"]

	in := DescribeAgentFeelingsV1Input{
		Persona:    p,
		Statements: statements,
	}

	var out DescribeAgentFeelingsV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Feelings
}

// GeneratePlanningNote implements llm.Cognition.
func (c *Client) GeneratePlanningNote(p llm.Persona, statements []string) string {
	prompt := prompts["describe_agent_feelings_v1"]

	in := ExtractSchedulingInformationV1Input{
		Persona:     p,
		Statements:  statements,
		CurrentDate: p.CurrentTime().Format("Monday January 02"),
	}

	var out ExtractSchedulingInformationV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Memory
}

// GenerateCurrentPlans implements llm.Cognition.
func (c *Client) GenerateCurrentPlans(p llm.Persona, plans string, thoughts string) string {
	prompt := prompts["generate_currently_v1"]

	in := GenerateCurrentlyV1Input{
		Persona:       p,
		CurrentDate:   p.CurrentTime().Format("Monday January 02"),
		YesterdayDate: p.CurrentTime().Add(-time.Hour * 24).Format("Monday January 02"),
		PlanningNote:  plans,
		ThoughtNote:   thoughts,
	}

	var out GenerateCurrentlyV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Status
}

func (c *Client) GenerateNewDailyRequirements(p llm.Persona) string {
	prompt := prompts["revise_daily_requirements_v1"]

	in := ReviseDailyRequirementsV1Input{
		Persona:     p,
		CurrentDate: p.CurrentTime().Format("Monday January 02"),
	}

	var out ReviseDailyRequirementsV1Output
	if err := c.doRequestWithRetry(context.Background(), prompt, in, &out, nil); err != nil {
		panic(fmt.Sprintf("could not perform request: %v", err))
	}

	return out.Day
}
