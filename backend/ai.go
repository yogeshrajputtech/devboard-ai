// AI endpoints — the only part of this backend that talks to Claude. Kept in
// its own file so main.go reads the same whether or not ANTHROPIC_API_KEY is
// configured: every handler here checks aiEnabled first and degrades to a
// 503 without ever dialing out, so plain CRUD is never affected by an AI
// outage or a missing key.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gin-gonic/gin"
)

var (
	aiClient  anthropic.Client
	aiEnabled bool
	aiModel   string
	aiTimeout time.Duration
)

func initAI() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	aiModel = env("ANTHROPIC_MODEL", "claude-opus-5")

	timeoutSec, err := strconv.Atoi(env("ANTHROPIC_TIMEOUT_SECONDS", "20"))
	if err != nil || timeoutSec <= 0 {
		timeoutSec = 20
	}
	aiTimeout = time.Duration(timeoutSec) * time.Second

	if apiKey == "" {
		log.Println("[backend] WARNING: ANTHROPIC_API_KEY not set — AI features disabled")
		aiEnabled = false
		return
	}

	aiClient = anthropic.NewClient(option.WithAPIKey(apiKey), option.WithRequestTimeout(aiTimeout))
	aiEnabled = true
	log.Printf("[backend] AI enabled (model=%s)", aiModel)
}

// setAIForTest lets ai_test.go point the client at a stub server and force
// aiEnabled without touching the environment. Same package, so unexported.
func setAIForTest(client anthropic.Client, enabled bool) {
	aiClient = client
	aiEnabled = enabled
	if aiTimeout == 0 {
		aiTimeout = 5 * time.Second
	}
}

func registerAIRoutes(r *gin.Engine) {
	r.POST("/ai/generate-tasks", aiGenerateTasks)
	r.POST("/ai/suggest-priority", aiSuggestPriority)
	r.POST("/ai/search", aiSearch)
	r.POST("/tasks/:id/ai/improve-description", aiImproveDescription)
	r.POST("/tasks/:id/ai/suggest-subtasks", aiSuggestSubtasks)
	r.POST("/tasks/:id/ai/summarize", aiSummarize)
}

// --- low-level Claude call helpers ---

var (
	errAIEmptyResponse = errors.New("ai: empty or non-text response")
	errAIInvalidJSON   = errors.New("ai: invalid json response")
)

// callClaudeJSON makes a single non-streaming, low-effort, thinking-disabled
// call constrained to `schema` via structured outputs (output_config.format)
// — this guarantees valid JSON matching the schema, so callers don't need to
// defensively strip markdown fences or retry on malformed output.
func callClaudeJSON(ctx context.Context, systemPrompt, userPrompt string, schema map[string]any, out any) error {
	resp, err := aiClient.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     aiModel,
		MaxTokens: 2048,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt))},
		Thinking:  anthropic.ThinkingConfigParamUnion{OfDisabled: disabledThinking()},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: schema},
		},
	})
	if err != nil {
		return err
	}
	text, err := firstText(resp)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(text), out); err != nil {
		return fmt.Errorf("%w: %v", errAIInvalidJSON, err)
	}
	return nil
}

// callClaudeText is the same call shape without a schema, for the two
// plain-text features (improve-description, summarize).
func callClaudeText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	resp, err := aiClient.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     aiModel,
		MaxTokens: 2048,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt))},
		Thinking:  anthropic.ThinkingConfigParamUnion{OfDisabled: disabledThinking()},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
		},
	})
	if err != nil {
		return "", err
	}
	return firstText(resp)
}

func disabledThinking() *anthropic.ThinkingConfigDisabledParam {
	d := anthropic.NewThinkingConfigDisabledParam()
	return &d
}

func firstText(resp *anthropic.Message) (string, error) {
	if len(resp.Content) == 0 {
		return "", errAIEmptyResponse
	}
	block := resp.Content[0]
	if block.Type != "text" {
		return "", errAIEmptyResponse
	}
	return block.Text, nil
}

// handleAIError maps a callClaude* error to the right HTTP response, logging
// full detail server-side only (same "log detail, return generic message"
// philosophy as fail() in main.go).
func handleAIError(c *gin.Context, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		log.Printf("[backend] AI TIMEOUT: %v", err)
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "ai_timeout"})
		return
	}
	if errors.Is(err, errAIEmptyResponse) || errors.Is(err, errAIInvalidJSON) {
		log.Printf("[backend] AI ERROR: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "ai_invalid_response"})
		return
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		log.Printf("[backend] AI ERROR: upstream status=%d: %v", apiErr.StatusCode, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "ai_upstream_error"})
		return
	}
	log.Printf("[backend] AI ERROR: %v", err)
	c.JSON(http.StatusBadGateway, gin.H{"error": "ai_upstream_error"})
}

func aiUnavailable(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ai_unavailable"})
}

func loadTask(id int) (Task, error) {
	row := db.QueryRow(taskSelect+` WHERE id = $1`, id)
	return scanTask(row)
}

func aiContext(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), aiTimeout)
}

// --- (1) AI task generation ---

func aiGenerateTasks(c *gin.Context) {
	if !aiEnabled {
		aiUnavailable(c)
		return
	}
	var body struct {
		ProjectID int    `json:"project_id"`
		Prompt    string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ProjectID == 0 || body.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id and prompt are required"})
		return
	}

	ctx, cancel := aiContext(c)
	defer cancel()

	system, user, schema := buildGenerateTasksPrompt(body.Prompt)
	var result struct {
		Tasks []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
		} `json:"tasks"`
	}
	if err := callClaudeJSON(ctx, system, user, schema, &result); err != nil {
		handleAIError(c, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		fail(c, err)
		return
	}
	defer tx.Rollback()

	created := make([]Task, 0, len(result.Tasks))
	for _, t := range result.Tasks {
		task, err := insertTask(tx, t.Title, t.Description, body.ProjectID, "todo", t.Priority, nil)
		if err != nil {
			fail(c, err)
			return
		}
		created = append(created, task)
	}
	if err := tx.Commit(); err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"tasks": created})
}

// --- (2)-(4) Smart task assistant — suggestion-only, no DB writes ---

func aiImproveDescription(c *gin.Context) {
	if !aiEnabled {
		aiUnavailable(c)
		return
	}
	task, ok := loadTaskParam(c)
	if !ok {
		return
	}

	ctx, cancel := aiContext(c)
	defer cancel()

	system, user := buildImproveDescriptionPrompt(task.Title, task.Description)
	text, err := callClaudeText(ctx, system, user)
	if err != nil {
		handleAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestion": text})
}

func aiSuggestSubtasks(c *gin.Context) {
	if !aiEnabled {
		aiUnavailable(c)
		return
	}
	task, ok := loadTaskParam(c)
	if !ok {
		return
	}

	ctx, cancel := aiContext(c)
	defer cancel()

	system, user, schema := buildSuggestSubtasksPrompt(task.Title, task.Description)
	var result struct {
		Subtasks []string `json:"subtasks"`
	}
	if err := callClaudeJSON(ctx, system, user, schema, &result); err != nil {
		handleAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"subtasks": result.Subtasks})
}

func aiSummarize(c *gin.Context) {
	if !aiEnabled {
		aiUnavailable(c)
		return
	}
	task, ok := loadTaskParam(c)
	if !ok {
		return
	}

	ctx, cancel := aiContext(c)
	defer cancel()

	system, user := buildSummarizePrompt(task.Title, task.Description)
	text, err := callClaudeText(ctx, system, user)
	if err != nil {
		handleAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": text})
}

// loadTaskParam reads :id, loads the task, and writes the appropriate error
// response itself on failure — callers just check the returned bool.
func loadTaskParam(c *gin.Context) (Task, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return Task{}, false
	}
	task, err := loadTask(id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return Task{}, false
	}
	if err != nil {
		fail(c, err)
		return Task{}, false
	}
	return task, true
}

// --- (5) Auto priority & tagging ---

func aiSuggestPriority(c *gin.Context) {
	if !aiEnabled {
		aiUnavailable(c)
		return
	}
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	ctx, cancel := aiContext(c)
	defer cancel()

	system, user, schema := buildSuggestPriorityPrompt(body.Title, body.Description)
	var result struct {
		Priority string   `json:"priority"`
		Tags     []string `json:"tags"`
	}
	if err := callClaudeJSON(ctx, system, user, schema, &result); err != nil {
		handleAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"priority": result.Priority, "tags": result.Tags})
}

// --- (6) Natural-language search ---

var (
	validTaskStatus   = map[string]bool{"todo": true, "in_progress": true, "blocked": true, "done": true}
	validTaskPriority = map[string]bool{"low": true, "medium": true, "high": true}
)

func aiSearch(c *gin.Context) {
	if !aiEnabled {
		aiUnavailable(c)
		return
	}
	var body struct {
		ProjectID int    `json:"project_id"`
		Query     string `json:"query"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ProjectID == 0 || body.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id and query are required"})
		return
	}

	ctx, cancel := aiContext(c)
	defer cancel()

	today := time.Now().Format("2006-01-02")
	system, user, schema := buildNLSearchPrompt(body.Query, today)
	var filter struct {
		Status    *string `json:"status"`
		Priority  *string `json:"priority"`
		DueBefore *string `json:"due_before"`
		DueAfter  *string `json:"due_after"`
		Keyword   *string `json:"keyword"`
	}
	if err := callClaudeJSON(ctx, system, user, schema, &filter); err != nil {
		handleAIError(c, err)
		return
	}

	// Validate every field against known enums/formats before it touches SQL
	// — same whitelist-style discipline as the PATCH handler in main.go.
	// A field that doesn't validate is dropped rather than erroring the
	// whole request, so a partially-confused model response still degrades
	// to a partial-but-safe filter.
	if filter.Status != nil && !validTaskStatus[*filter.Status] {
		filter.Status = nil
	}
	if filter.Priority != nil && !validTaskPriority[*filter.Priority] {
		filter.Priority = nil
	}
	if filter.DueBefore != nil {
		if _, err := time.Parse("2006-01-02", *filter.DueBefore); err != nil {
			filter.DueBefore = nil
		}
	}
	if filter.DueAfter != nil {
		if _, err := time.Parse("2006-01-02", *filter.DueAfter); err != nil {
			filter.DueAfter = nil
		}
	}

	where := []string{"project_id = $1"}
	args := []interface{}{body.ProjectID}
	i := 2
	if filter.Status != nil {
		where = append(where, "status = $"+strconv.Itoa(i))
		args = append(args, *filter.Status)
		i++
	}
	if filter.Priority != nil {
		where = append(where, "priority = $"+strconv.Itoa(i))
		args = append(args, *filter.Priority)
		i++
	}
	if filter.DueBefore != nil {
		where = append(where, "due_date < $"+strconv.Itoa(i))
		args = append(args, *filter.DueBefore)
		i++
	}
	if filter.DueAfter != nil {
		where = append(where, "due_date > $"+strconv.Itoa(i))
		args = append(args, *filter.DueAfter)
		i++
	}
	if filter.Keyword != nil && *filter.Keyword != "" {
		idx := strconv.Itoa(i)
		where = append(where, "(title ILIKE '%' || $"+idx+" || '%' OR description ILIKE '%' || $"+idx+" || '%')")
		args = append(args, *filter.Keyword)
		i++
	}

	query := taskSelect + " WHERE " + join(where, " AND ") + " ORDER BY id"
	rows, err := db.Query(query, args...)
	if err != nil {
		fail(c, err)
		return
	}
	defer rows.Close()
	tasks, err := scanTasks(rows)
	if err != nil {
		fail(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": tasks, "filter": filter})
}
