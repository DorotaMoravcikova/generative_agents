package openai

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/fvdveen/generative_agents/simulation_server/memory"
	"github.com/xeipuuv/gojsonschema"

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
	name       string
	schema     schema
	template   *template.Template
	jsonSchema *gojsonschema.Schema
}

func (p prompt) validateJSON(json string) ([]gojsonschema.ResultError, bool, error) {
	doc := gojsonschema.NewStringLoader(json)

	res, err := p.jsonSchema.Validate(doc)
	if err != nil {
		return nil, false, fmt.Errorf("json schema validation failed: %w", err)
	}

	return res.Errors(), res.Valid(), nil
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

		jsonSchema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(content))
		if err != nil {
			panic(fmt.Sprintf("Could not create json schema: %v", err))
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

		prompts[name] = prompt{name, schema, template, jsonSchema}
	}

	return prompts
}

var prompts = loadPrompts()

type attemptResult struct {
	content      string
	inputTokens  int
	outputTokens int
	totalTokens  int
}

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

	textModel          string
	embeddingModel     string
	maxRetries         int
	useChatCompletions bool

	llmSeq atomic.Uint64
}

func New(opts ...ClientOpt) *Client {
	client := &Client{textModel: "gpt-5-nano", embeddingModel: "text-embedding-ada-002", maxRetries: 8, logger: slog.Default()}

	for _, opt := range opts {
		opt(client)
	}

	openaiOpts := []option.RequestOption{option.WithAPIKey(client.apiKey)}
	if client.url != "" {
		openaiOpts = append(openaiOpts, option.WithBaseURL(client.url))
	}
	client.client = openai.NewClient(openaiOpts...)
	client.useChatCompletions = client.url != ""

	return client
}

func (c *Client) newID() string {
	n := c.llmSeq.Add(1)
	return fmt.Sprintf("llm-%d", n)
}

func (c *Client) responseParams(input responses.ResponseNewParamsInputUnion, schema schema) responses.ResponseNewParams {
	var r responses.ResponseNewParams

	if c.textModel == "gpt-5-nano" {
		r = responses.ResponseNewParams{
			Model:     c.textModel,
			Reasoning: shared.ReasoningParam{Effort: "low"},
			Input:     input,
			Text: responses.ResponseTextConfigParam{
				Format: responses.ResponseFormatTextConfigParamOfJSONSchema(schema.Name, schema.Schema),
			},
		}
	} else {
		r = responses.ResponseNewParams{
			Model:     c.textModel,
			Reasoning: shared.ReasoningParam{Effort: "medium"},
			Input:     input,
			Text: responses.ResponseTextConfigParam{
				Format: responses.ResponseFormatTextConfigParamOfJSONSchema(schema.Name, schema.Schema),
			},
			Temperature: param.NewOpt(0.5),
			TopP:        param.NewOpt(0.9),
		}
	}

	return r
}

func inputMsg(role responses.EasyInputMessageRole, content string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemUnionParam{
		OfMessage: &responses.EasyInputMessageParam{
			Role: role,
			Type: responses.EasyInputMessageTypeMessage,
			Content: responses.EasyInputMessageContentUnionParam{
				OfString: param.NewOpt(content),
			},
		},
	}
}

func buildRetryMsg(errMsgs []string) string {
	var sb strings.Builder
	sb.WriteString("The generated response was invalid. Please fix the following errors and return only valid JSON:\n")
	for _, e := range errMsgs {
		sb.WriteString("- ")
		sb.WriteString(e)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func withResultAttrs(log *slog.Logger, r attemptResult) *slog.Logger {
	return log.With(
		"input_tokens", r.inputTokens,
		"output_tokens", r.outputTokens,
		"total_tokens", r.totalTokens,
		"response_hash", hashString(r.content),
		"response_len", len(r.content),
	)
}

// tryUnmarshal tries direct unmarshal, falling back to extracting JSON from surrounding text.
func tryUnmarshal(text string, output any) error {
	if err := json.Unmarshal([]byte(text), output); err != nil {
		extracted := extractJSON(text)
		if extracted != text {
			if err2 := json.Unmarshal([]byte(extracted), output); err2 == nil {
				return nil
			}
		}
		return fmt.Errorf("could not unmarshal json: %w", err)
	}
	return nil
}

func (c *Client) doRequestWithRetry(ctx context.Context, prompt prompt, params any, output any, validationFn func() error) error {
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

	if c.useChatCompletions {
		return c.doRequestWithRetryChat(ctx, prompt, promptText, output, validationFn, log, start)
	}
	return c.doRequestWithRetryResponses(ctx, prompt, promptText, output, validationFn, log, start)
}

func (c *Client) doRequestWithRetryResponses(ctx context.Context, p prompt, promptText string, output any, validationFn func() error, log *slog.Logger, start time.Time) error {
	conversation := responses.ResponseInputParam{
		inputMsg(responses.EasyInputMessageRoleUser, promptText),
	}

	var lastResult attemptResult
	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		resp, err := c.client.Responses.New(ctx, c.responseParams(
			responses.ResponseNewParamsInputUnion{OfInputItemList: conversation}, p.schema))
		if err != nil {
			lastErr = err
			log.Error("llm_call_fail",
				"type", "llm_call",
				"phase", "fail",
				"attempt", attempt+1,
				"total_latency", time.Since(start),
				"err", err,
			)
			return err
		}

		result := attemptResult{
			content:      resp.OutputText(),
			inputTokens:  int(resp.Usage.InputTokens),
			outputTokens: int(resp.Usage.OutputTokens),
			totalTokens:  int(resp.Usage.TotalTokens),
		}
		lastResult = result
		l := withResultAttrs(log, result)

		appendOutputItems := func() {
			for _, item := range resp.Output {
				switch item.Type {
				case "message":
					msg := item.AsMessage().ToParam()
					conversation = append(conversation, responses.ResponseInputItemUnionParam{OfOutputMessage: &msg})
				case "reasoning":
					rs := item.AsReasoning().ToParam()
					conversation = append(conversation, responses.ResponseInputItemUnionParam{OfReasoning: &rs})
				}
			}
		}

		if unmarshalErr := tryUnmarshal(result.content, output); unmarshalErr != nil {
			lastErr = unmarshalErr
			appendOutputItems()
			conversation = append(conversation, inputMsg(responses.EasyInputMessageRoleUser, buildRetryMsg([]string{
				unmarshalErr.Error(),
				"Hint: only return a valid JSON object, _DO NOT_ include surrounding markdown or text",
			})))
			l.Warn("llm_retry",
				slog.String("phase", "retry"),
				slog.Int("attempt", attempt+1),
				slog.String("reason", "json_unmarshal"),
				slog.Any("err", unmarshalErr),
			)
			continue
		}

		errs, valid, err := p.validateJSON(extractJSON(result.content))
		if err != nil {
			l.Error("llm_json_validation_error",
				"type", "llm_call",
				"phase", "validation",
				"attempt", attempt+1,
				"total_latency", time.Since(start),
				"err", err,
			)
		}

		if !valid {
			errMsgs := make([]string, len(errs))
			for i, e := range errs {
				errMsgs[i] = fmt.Sprintf("%s: %s", e.Field(), e.Description())
			}
			appendOutputItems()
			conversation = append(conversation, inputMsg(responses.EasyInputMessageRoleUser, buildRetryMsg(errMsgs)))
			l.Warn("llm_retry",
				slog.String("phase", "retry"),
				slog.Int("attempt", attempt+1),
				slog.String("reason", "json_validation"),
				slog.Int("validation_error_count", len(errs)),
				slog.Any("validation_errors", validationSlogIssues(errs)),
			)
			continue
		}

		if validationFn != nil {
			if err := validationFn(); err != nil {
				lastErr = err
				appendOutputItems()
				conversation = append(conversation, inputMsg(responses.EasyInputMessageRoleUser, buildRetryMsg([]string{err.Error()})))
				l.Warn("llm_retry",
					"type", "llm_call",
					"phase", "retry",
					"attempt", attempt+1,
					"reason", "validation",
					"err", err,
					"response_hash", hashString(result.content),
					"response_len", len(result.content),
				)
				continue
			}
		}

		l.Info("llm_call_ok",
			"type", "llm_call",
			"phase", "ok",
			"attempts_total", attempt+1,
			"total_latency", time.Since(start),
			"response_hash", hashString(result.content),
			"response_len", len(result.content),
		)
		return nil
	}

	withResultAttrs(log, lastResult).With("output_raw", lastResult.content).Error("llm_call_fail",
		"type", "llm_call",
		"phase", "fail",
		"attempts_total", c.maxRetries,
		"total_latency", time.Since(start),
		"prompt_raw", promptText,
		"err", lastErr,
	)
	return fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

func (c *Client) doRequestWithRetryChat(ctx context.Context, p prompt, promptText string, output any, validationFn func() error, log *slog.Logger, start time.Time) error {
	conversation := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(promptText),
	}

	var lastResult attemptResult
	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModel(c.textModel),
			Messages: conversation,
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
					JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
						Name:   p.schema.Name,
						Schema: p.schema.Schema,
						Strict: param.NewOpt(true),
					},
				},
			},
			Temperature: param.NewOpt(0.5),
			TopP:        param.NewOpt(0.9),
		})
		if err != nil {
			lastErr = err
			log.Error("llm_call_fail",
				"type", "llm_call",
				"phase", "fail",
				"attempt", attempt+1,
				"total_latency", time.Since(start),
				"err", err,
			)
			return err
		}

		result := attemptResult{
			content:      resp.Choices[0].Message.Content,
			inputTokens:  int(resp.Usage.PromptTokens),
			outputTokens: int(resp.Usage.CompletionTokens),
			totalTokens:  int(resp.Usage.TotalTokens),
		}
		lastResult = result
		l := withResultAttrs(log, result)

		if unmarshalErr := tryUnmarshal(result.content, output); unmarshalErr != nil {
			lastErr = unmarshalErr
			conversation = append(conversation,
				openai.AssistantMessage(result.content),
				openai.UserMessage(buildRetryMsg([]string{
					unmarshalErr.Error(),
					"Hint: only return a valid JSON object, _DO NOT_ include surrounding markdown or text",
				})),
			)
			l.Warn("llm_retry",
				slog.String("phase", "retry"),
				slog.Int("attempt", attempt+1),
				slog.String("reason", "json_unmarshal"),
				slog.Any("err", unmarshalErr),
			)
			continue
		}

		errs, valid, err := p.validateJSON(extractJSON(result.content))
		if err != nil {
			l.Error("llm_json_validation_error",
				"type", "llm_call",
				"phase", "validation",
				"attempt", attempt+1,
				"total_latency", time.Since(start),
				"err", err,
			)
		}

		if !valid {
			errMsgs := make([]string, len(errs))
			for i, e := range errs {
				errMsgs[i] = fmt.Sprintf("%s: %s", e.Field(), e.Description())
			}
			conversation = append(conversation,
				openai.AssistantMessage(result.content),
				openai.UserMessage(buildRetryMsg(errMsgs)),
			)
			l.Warn("llm_retry",
				slog.String("phase", "retry"),
				slog.Int("attempt", attempt+1),
				slog.String("reason", "json_validation"),
				slog.Int("validation_error_count", len(errs)),
				slog.Any("validation_errors", validationSlogIssues(errs)),
			)
			continue
		}

		if validationFn != nil {
			if err := validationFn(); err != nil {
				lastErr = err
				conversation = append(conversation,
					openai.AssistantMessage(result.content),
					openai.UserMessage(buildRetryMsg([]string{err.Error()})),
				)
				l.Warn("llm_retry",
					"type", "llm_call",
					"phase", "retry",
					"attempt", attempt+1,
					"reason", "validation",
					"err", err,
					"response_hash", hashString(result.content),
					"response_len", len(result.content),
				)
				continue
			}
		}

		l.Info("llm_call_ok",
			"type", "llm_call",
			"phase", "ok",
			"attempts_total", attempt+1,
			"total_latency", time.Since(start),
			"response_hash", hashString(result.content),
			"response_len", len(result.content),
		)
		return nil
	}

	withResultAttrs(log, lastResult).With("output_raw", lastResult.content).Error("llm_call_fail",
		"type", "llm_call",
		"phase", "fail",
		"attempts_total", c.maxRetries,
		"total_latency", time.Since(start),
		"prompt_raw", promptText,
		"err", lastErr,
	)
	return fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

func extractJSON(s string) string {
	start := strings.IndexAny(s, "{[")
	if start == -1 {
		return s
	}
	var close byte = '}'
	if s[start] == '[' {
		close = ']'
	}
	end := strings.LastIndexByte(s, close)
	if end <= start {
		return s
	}
	return s[start : end+1]
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	// 8–12 bytes is plenty for logging fingerprints
	return hex.EncodeToString(sum[:8])
}

func validationSlogIssues(errs []gojsonschema.ResultError) slog.Value {
	attrs := make([]slog.Attr, 0, len(errs))

	for _, e := range errs {
		path := e.Field()
		if path == "" {
			path = "(root)"
		}

		attrs = append(attrs, slog.Group(
			"issue",
			slog.String("path", path),
			slog.String("message", e.Description()),
			slog.Any("details", e.Details()),
		))
	}

	return slog.GroupValue(attrs...)
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
