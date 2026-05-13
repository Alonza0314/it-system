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

type NfPrResult struct {
	NfName string
	PR     int
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
	switch status {
	case constant.TASK_STATUS_SUCCESS:
		return "✅"
	case constant.TASK_STATUS_TIMEOUT:
		return "❗"
	default:
		return "❌"
	}
}

func statusText(status string) string {
	switch status {
	case constant.TASK_STATUS_SUCCESS:
		return "Success"
	case constant.TASK_STATUS_TIMEOUT:
		return "Timeout"
	default:
		return "Failed"
	}
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

func formatNfPrDisplayName(nfPr NfPrResult) string {
	nfName := strings.TrimSpace(nfPr.NfName)
	if nfName == "" {
		nfName = "unknown"
	}

	return fmt.Sprintf("%s #%d", nfName, nfPr.PR)
}

func formatNfPrSummary(nfPrList []NfPrResult) string {
	if len(nfPrList) == 0 {
		return "No PRs"
	}

	parts := make([]string, 0, len(nfPrList))
	for _, nfPr := range nfPrList {
		parts = append(parts, formatNfPrDisplayName(nfPr))
	}

	summary := strings.Join(parts, ", ")
	if len(summary) <= constant.MAX_THREAD_PR_SUMMARY_LENGTH {
		return summary
	}

	for count := len(parts) - 1; count > 0; count-- {
		remaining := len(parts) - count
		candidate := fmt.Sprintf("%s +%d more", strings.Join(parts[:count], ", "), remaining)
		if len(candidate) <= constant.MAX_THREAD_PR_SUMMARY_LENGTH {
			return candidate
		}
	}

	return fmt.Sprintf("%d PRs", len(parts))
}

func formatNfPrDetails(nfPrList []NfPrResult) string {
	if len(nfPrList) == 0 {
		return "- (No PRs)"
	}

	lines := make([]string, 0, len(nfPrList))
	for _, nfPr := range nfPrList {
		lines = append(lines, fmt.Sprintf("- %s", formatNfPrDisplayName(nfPr)))
	}

	return strings.Join(lines, "\n")
}

// SendTaskNotification posts a new thread to a Discord Forum channel via Webhook.
// userDiscordId is expected to be the Discord user ID (snowflake) for @mention to work.
func SendTaskNotification(webhookURL string, taskID uint64, username, userDiscordId, status string, pipelines []PipelineResult, nfPrList []NfPrResult) error {
	emoji := statusEmoji(status)
	statusUpper := strings.ToUpper(status)

	threadName := fmt.Sprintf("Task #%d · %s · %s · %s %s", taskID, formatNfPrSummary(nfPrList), username, emoji, statusUpper)
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
		"%s\n\nTask #%d is done.\nFetched PRs:\n%s\n\nResult: %s %s\n\nPipeline:\n%s",
		content,
		taskID,
		formatNfPrDetails(nfPrList),
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
