package main

import (
	"fmt"
	"strings"
)

func validateTaskFields(fields map[string]interface{}) error {
	for key, value := range fields {
		switch key {
		case "title":
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) == "" || len([]rune(text)) > 300 {
				return fmt.Errorf("title must be a nonempty string of at most 300 characters")
			}
		case "description":
			if value != nil {
				if _, ok := value.(string); !ok {
					return fmt.Errorf("description must be text or null")
				}
			}
		case "status":
			text, ok := value.(string)
			if !ok || text != "todo" && text != "in_progress" && text != "blocked" && text != "done" {
				return fmt.Errorf("invalid task status")
			}
		case "priority":
			text, ok := value.(string)
			if !ok || text != "low" && text != "medium" && text != "high" {
				return fmt.Errorf("invalid task priority")
			}
		}
	}
	return nil
}
