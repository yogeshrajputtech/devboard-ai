package main

import "fmt"

// Per-feature prompt + JSON-schema builders, kept separate from ai.go's
// HTTP/DB logic so prompt text is easy to review and tune on its own.
//
// Schemas use "anyOf": [{type...}, {type:"null"}] for nullable fields —
// the documented, supported way to express nullability in Claude's
// structured-output JSON Schema subset (a bare `"type": ["string","null"]`
// array is not on the supported-features list).

func nullableString(enum []string) map[string]any {
	strSchema := map[string]any{"type": "string"}
	if len(enum) > 0 {
		strSchema["enum"] = enum
	}
	return map[string]any{
		"anyOf": []any{strSchema, map[string]any{"type": "null"}},
	}
}

// --- (1) AI task generation ---

func buildGenerateTasksPrompt(goal string) (system, user string, schema map[string]any) {
	system = "You are a project-planning assistant for a developer task board. " +
		"Given a goal, break it into 3 to 8 concrete, actionable tasks. " +
		"Keep titles short (under 80 characters) and descriptions to 1-2 sentences."
	user = "Goal: " + goal
	schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tasks": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": 8,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":       map[string]any{"type": "string"},
						"description": map[string]any{"type": "string"},
						"priority":    map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
					},
					"required":             []string{"title", "description", "priority"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"tasks"},
		"additionalProperties": false,
	}
	return
}

// --- (2)-(4) Smart task assistant ---

func buildImproveDescriptionPrompt(title, description string) (system, user string) {
	system = "Rewrite this task's description to be clearer and more actionable, " +
		"while preserving its original intent. Respond with ONLY the improved " +
		"description as plain text — no preamble, no markdown, no quotes."
	user = fmt.Sprintf("Title: %s\nCurrent description: %s", title, orNone(description))
	return
}

func buildSuggestSubtasksPrompt(title, description string) (system, user string, schema map[string]any) {
	system = "Break this task down into 2 to 6 short, concrete subtask titles " +
		"(each under 80 characters) that together would complete it."
	user = fmt.Sprintf("Title: %s\nDescription: %s", title, orNone(description))
	schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subtasks": map[string]any{
				"type":     "array",
				"minItems": 2,
				"maxItems": 6,
				"items":    map[string]any{"type": "string"},
			},
		},
		"required":             []string{"subtasks"},
		"additionalProperties": false,
	}
	return
}

func buildSummarizePrompt(title, description string) (system, user string) {
	system = "Summarize this task in 1 to 3 sentences, suitable for a status update. " +
		"Respond with ONLY the summary as plain text — no preamble, no markdown."
	user = fmt.Sprintf("Title: %s\nDescription: %s", title, orNone(description))
	return
}

// --- (5) Auto priority & tagging ---

func buildSuggestPriorityPrompt(title, description string) (system, user string, schema map[string]any) {
	system = "Given a task's title and description, suggest a priority " +
		"(low, medium, or high) and 1 to 3 short (one or two word) tags."
	user = fmt.Sprintf("Title: %s\nDescription: %s", title, orNone(description))
	schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"priority": map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
			"tags": map[string]any{
				"type":     "array",
				"minItems": 1,
				"maxItems": 3,
				"items":    map[string]any{"type": "string"},
			},
		},
		"required":             []string{"priority", "tags"},
		"additionalProperties": false,
	}
	return
}

// --- (6) Natural-language search ---

func buildNLSearchPrompt(query, today string) (system, user string, schema map[string]any) {
	system = fmt.Sprintf(
		"Today's date is %s. Translate the user's natural-language task search "+
			"into a structured filter. Set any field you can't confidently derive to null. "+
			"Dates must be in YYYY-MM-DD format.",
		today,
	)
	user = "Query: " + query
	schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status":     nullableString([]string{"todo", "in_progress", "blocked", "done"}),
			"priority":   nullableString([]string{"low", "medium", "high"}),
			"due_before": nullableString(nil),
			"due_after":  nullableString(nil),
			"keyword":    nullableString(nil),
		},
		"required":             []string{"status", "priority", "due_before", "due_after", "keyword"},
		"additionalProperties": false,
	}
	return
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
