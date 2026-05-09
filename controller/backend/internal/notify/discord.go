package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Alonza0314/it-system/controller/backend/constant"
)

type PipelineResult struct {
	Name   string
	Status string
}

type discordPayload struct {
	ThreadName      string           `json:"thread_name"`
	Content         string           `json:"content"`
	AllowedMentions *allowedMentions `json:"allowed_mentions,omitempty"`
}

type allowedMentions struct {
	Parse []string `json:"parse,omitempty"`
	Users []string `json:"users,omitempty"`
}

func statusEmoji(status string) string {
	if status == "success" {
		return "✅"
	}
	return "❌"
}

func statusText(status string) string {
	if status == "success" {
		return "Success"
	}
	return "Failed"
}

func reorderPipelinesForDisplay(pipelines []PipelineResult) []PipelineResult {
	ordered := make([]PipelineResult, 0, len(pipelines))
	tests := make([]PipelineResult, 0, len(pipelines))
	var prepare, fetch, makeNF, cleanup *PipelineResult

	for i := range pipelines {
		p := pipelines[i]
		switch p.Name {
		case constant.TESTCASE_PREPARE_FREE5GC:
			cp := p
			prepare = &cp
		case constant.TESTCASE_FETCH_PRS:
			cp := p
			fetch = &cp
		case constant.TESTCASE_MAKE_NF:
			cp := p
			makeNF = &cp
		case constant.TESTCASE_CLEANUP:
			cp := p
			cleanup = &cp
		default:
			tests = append(tests, p)
		}
	}

	if prepare != nil {
		ordered = append(ordered, *prepare)
	}
	if fetch != nil {
		ordered = append(ordered, *fetch)
	}
	if makeNF != nil {
		ordered = append(ordered, *makeNF)
	}

	ordered = append(ordered, tests...)

	if cleanup != nil {
		ordered = append(ordered, *cleanup)
	}

	return ordered
}

// SendTaskNotification posts a new thread to a Discord Forum channel via Webhook.
// userDiscordId is expected to be the Discord user ID (snowflake) for @mention to work.
func SendTaskNotification(webhookURL string, taskID uint64, userDiscordId, status string, pipelines []PipelineResult) error {
	emoji := statusEmoji(status)
	statusUpper := strings.ToUpper(status)

	threadName := fmt.Sprintf("Task #%d · %s · %s %s", taskID, userDiscordId, emoji, statusUpper)
	trimmedUsername := strings.TrimSpace(userDiscordId)
	content := fmt.Sprintf("@%s Task Finished!", trimmedUsername)
	var mentions *allowedMentions
	if userID, ok := extractDiscordUserID(trimmedUsername); ok {
		content = fmt.Sprintf("<@%s> Task Finished!", userID)
		mentions = &allowedMentions{Users: []string{userID}}
	}

	pipelines = reorderPipelinesForDisplay(pipelines)

	var detailLines []string
	for _, p := range pipelines {
		detailLines = append(detailLines, fmt.Sprintf("- %s %s", statusEmoji(p.Status), p.Name))
	}
	details := strings.Join(detailLines, "\n")
	if details == "" {
		details = "- (No test items)"
	}

	content = fmt.Sprintf(
		"%s\n\nTask #%d is done.\nResult: %s %s\n\nPipeline:\n%s",
		content,
		taskID,
		emoji,
		statusText(status),
		details,
	)

	payload := discordPayload{
		ThreadName:      threadName,
		Content:         content,
		AllowedMentions: mentions,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: failed to marshal payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: HTTP request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: unexpected status code %d", resp.StatusCode)
	}

	return nil
}

func extractDiscordUserID(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}

	if strings.HasPrefix(trimmed, "<@") && strings.HasSuffix(trimmed, ">") {
		trimmed = strings.TrimSuffix(strings.TrimPrefix(trimmed, "<@"), ">")
		trimmed = strings.TrimPrefix(trimmed, "!")
	}

	_, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return "", false
	}

	return trimmed, true
}
