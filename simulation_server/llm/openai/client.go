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
	client := &Client{textModel: "gpt-5-nano", embeddingModel: "text-embedding-ada-002", maxRetries: 8, logger: slog.Default()}

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
			Reasoning: shared.ReasoningParam{Effort: "low"},
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

func outputMsg(id, text string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemUnionParam{
		OfOutputMessage: &responses.ResponseOutputMessageParam{
			ID:     id,
			Status: responses.ResponseOutputMessageStatusCompleted,
			Content: []responses.ResponseOutputMessageContentUnionParam{
				{
					OfOutputText: &responses.ResponseOutputTextParam{
						Text:        text,
						Annotations: []responses.ResponseOutputTextAnnotationUnionParam{},
					},
				},
			},
		},
	}
}

func appendRetryMessages(current responses.ResponseInputParam, id, badResponse string, errMsgs []string) responses.ResponseInputParam {
	var sb strings.Builder
	sb.WriteString("The generated response was invalid. Please fix the following errors and return only valid JSON:\n")
	for _, e := range errMsgs {
		sb.WriteString("- ")
		sb.WriteString(e)
		sb.WriteByte('\n')
	}

	return append(current,
		outputMsg(id, badResponse),
		inputMsg(responses.EasyInputMessageRoleUser, sb.String()),
	)
}

func (c *Client) doRequest(ctx context.Context, input responses.ResponseNewParamsInputUnion, schema schema, output any) (*responses.Response, error) {
	resp, err := c.client.Responses.New(ctx, c.responseParams(input, schema))
	if err != nil {
		return resp, fmt.Errorf("could not execute prompt: %w", err)
	}

	raw := resp.OutputText()

	if err := json.Unmarshal([]byte(raw), output); err != nil {
		// The model may have wrapped the JSON in surrounding text; try extracting
		// the content between the first { or [ and the last } or ]
		extracted := extractJSON(raw)
		if extracted != raw {
			if err2 := json.Unmarshal([]byte(extracted), output); err2 == nil {
				return resp, nil
			}
		}
		return resp, fmt.Errorf("could not unmarshal json: %w", err)
	}

	return resp, nil
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
	var lastResp *responses.Response

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
	conversation := responses.ResponseInputParam{
		inputMsg(responses.EasyInputMessageRoleUser, promptText),
	}
	currentInput := responses.ResponseNewParamsInputUnion{
		OfInputItemList: conversation,
	}
	var resp *responses.Response
	var err error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		resp, err = c.doRequest(ctx, currentInput, prompt.schema, output)
		lastResp = resp

		l := log
		if resp != nil {
			l = l.With(
				"input_tokens", int(resp.Usage.InputTokens),
				"output_tokens", int(resp.Usage.OutputTokens),
				"total_tokens", int(resp.Usage.TotalTokens),
				"response_hash", hashString(resp.OutputText()),
				"response_len", len(resp.OutputText()),
			)
		}

		if err != nil {
			lastErr = err

			// retry on JSON unmarshalling errors, feeding the bad response + error back
			if isJSONUnmarshalError(err) && resp != nil {
				conversation = appendRetryMessages(conversation, fmt.Sprintf("resp_%d", attempt), resp.OutputText(), []string{err.Error(), "Hint: only return a valid JSON object, _DO NOT_ include surrounding markdown or text"})
				currentInput = responses.ResponseNewParamsInputUnion{OfInputItemList: conversation}
				l.Warn("llm_retry",
					slog.String("phase", "retry"),
					slog.Int("attempt", attempt+1),
					slog.String("reason", "json_unmarshal"),
					slog.Any("err", err),
				)
				continue
			}

			l.Error("llm_call_fail",
				"type", "llm_call",
				"phase", "fail",
				"attempt", attempt+1,
				"total_latency", time.Since(start),
				"err", err,
			)
			return err
		}

		errs, valid, err := prompt.validateJSON(resp.OutputText())
		if err != nil {
			l.Error("llm_json_validation_error",
				"type", "llm_call",
				"phase", "validation",
				"attempt", attempt+1,
				"total_latency", time.Since(start),
				"err", err,
			)
		}

		// retry on schema validation failure, feeding errors back to the model
		if !valid {
			errMsgs := make([]string, len(errs))
			for i, e := range errs {
				errMsgs[i] = fmt.Sprintf("%s: %s", e.Field(), e.Description())
			}
			conversation = appendRetryMessages(conversation, fmt.Sprintf("resp_%d", attempt), resp.OutputText(), errMsgs)
			currentInput = responses.ResponseNewParamsInputUnion{OfInputItemList: conversation}
			l.Warn("llm_retry",
				slog.String("phase", "retry"),
				slog.Int("attempt", attempt+1),
				slog.String("reason", "json_validation"),
				slog.Int("validation_error_count", len(errs)),
				slog.Any("validation_errors", validationSlogIssues(errs)),
			)
			continue
		}

		// If validation function is provided, run it and retry on failure
		if validationFn != nil {
			if err := validationFn(); err != nil {
				lastErr = err
				conversation = appendRetryMessages(conversation, fmt.Sprintf("resp_%d", attempt), resp.OutputText(), []string{err.Error()})
				currentInput = responses.ResponseNewParamsInputUnion{OfInputItemList: conversation}
				l.Warn("llm_retry",
					"type", "llm_call",
					"phase", "retry",
					"attempt", attempt+1,
					"reason", "validation",
					"err", err,
					"response_hash", hashString(resp.OutputText()),
					"response_len", len(resp.OutputText()),
				)
				continue
			}
		}

		l.Info("llm_call_ok",
			"type", "llm_call",
			"phase", "ok",
			"attempts_total", attempt+1,
			"total_latency", time.Since(start),
			"response_hash", hashString(resp.OutputText()),
			"response_len", len(resp.OutputText()),
		)
		// Success
		return nil
	}

	l := log
	if resp != nil {
		l = l.With(
			slog.Int("input_tokens", int(resp.Usage.InputTokens)),
			slog.Int("output_tokens", int(resp.Usage.OutputTokens)),
			slog.Int("total_tokens", int(resp.Usage.TotalTokens)),
		)
	}

	if lastResp != nil {
		l = l.With("output_raw", lastResp.OutputText())
	}

	l.Error("llm_call_fail",
		"type", "llm_call",
		"phase", "fail",
		"attempts_total", c.maxRetries,
		"total_latency", time.Since(start),
		"prompt_raw", promptText,
		"err", lastErr,
	)

	return fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
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
