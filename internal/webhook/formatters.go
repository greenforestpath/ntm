package webhook

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type BuiltInFormat string

const (
	BuiltInFormatJSON    BuiltInFormat = "json"
	BuiltInFormatSlack   BuiltInFormat = "slack"
	BuiltInFormatDiscord BuiltInFormat = "discord"
	BuiltInFormatTeams   BuiltInFormat = "teams"
)

func normalizeBuiltInFormat(format string) BuiltInFormat {
	s := strings.ToLower(strings.TrimSpace(format))
	switch s {
	case "json":
		return BuiltInFormatJSON
	case "slack":
		return BuiltInFormatSlack
	case "discord":
		return BuiltInFormatDiscord
	case "teams", "msteams", "ms-teams", "microsoft-teams", "microsoft_teams":
		return BuiltInFormatTeams
	default:
		return BuiltInFormat(s)
	}
}

func buildBuiltInPayload(event Event, format string) ([]byte, error) {
	switch normalizeBuiltInFormat(format) {
	case BuiltInFormatJSON:
		return json.Marshal(event)
	case BuiltInFormatSlack:
		return json.Marshal(buildSlackPayload(event))
	case BuiltInFormatDiscord:
		return json.Marshal(buildDiscordPayload(event))
	case BuiltInFormatTeams:
		return json.Marshal(buildTeamsPayload(event))
	default:
		return nil, fmt.Errorf("unknown webhook format %q (supported: json, slack, discord, teams)", strings.TrimSpace(format))
	}
}

type eventSeverity string

const (
	severityInfo    eventSeverity = "info"
	severitySuccess eventSeverity = "success"
	severityWarning eventSeverity = "warning"
	severityError   eventSeverity = "error"
)

func classifySeverity(eventType string) eventSeverity {
	t := strings.ToLower(strings.TrimSpace(eventType))
	switch {
	case t == "":
		return severityInfo
	case strings.Contains(t, "error"), strings.Contains(t, "failed"), strings.Contains(t, "crash"), strings.Contains(t, "panic"):
		return severityError
	case strings.Contains(t, "warn"), strings.Contains(t, "degrad"), strings.Contains(t, "rate_limit"), strings.Contains(t, "rate-limit"):
		return severityWarning
	case strings.Contains(t, "success"), strings.Contains(t, "complete"), strings.Contains(t, "done"), strings.Contains(t, "healthy"):
		return severitySuccess
	default:
		return severityInfo
	}
}

func formatTitle(event Event) string {
	t := strings.TrimSpace(event.Type)
	if t == "" {
		return "NTM Event"
	}
	return "NTM: " + t
}

func formatSummary(event Event) string {
	msg := strings.TrimSpace(event.Message)
	if msg == "" {
		msg = "(no message)"
	}
	return msg
}

func formatRFC3339(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func sortedKV(m map[string]string) [][2]string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([][2]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, [2]string{k, m[k]})
	}
	return out
}

// =============================================================================
// Slack
// =============================================================================

type slackText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

type slackBlock struct {
	Type   string      `json:"type"`
	Text   *slackText  `json:"text,omitempty"`
	Fields []slackText `json:"fields,omitempty"`
	// Context block supports "elements", but we keep it simple via fields.
}

type slackPayload struct {
	Text   string       `json:"text"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

func buildSlackPayload(event Event) slackPayload {
	title := formatTitle(event)
	summary := formatSummary(event)

	fields := make([]slackText, 0, 4)
	if event.Session != "" {
		fields = append(fields, slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Session:* %s", event.Session)})
	}
	if event.Agent != "" {
		fields = append(fields, slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Agent:* %s", event.Agent)})
	}
	if event.Pane != "" {
		fields = append(fields, slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Pane:* %s", event.Pane)})
	}
	if ts := formatRFC3339(event.Timestamp); ts != "" {
		fields = append(fields, slackText{Type: "mrkdwn", Text: fmt.Sprintf("*Time:* %s", ts)})
	}

	blocks := []slackBlock{
		{
			Type: "header",
			Text: &slackText{Type: "plain_text", Text: title, Emoji: true},
		},
		{
			Type: "section",
			Text: &slackText{Type: "mrkdwn", Text: summary},
		},
	}
	if len(fields) > 0 {
		blocks = append(blocks, slackBlock{Type: "section", Fields: fields})
	}

	kv := sortedKV(event.Details)
	if len(kv) > 0 {
		var sb strings.Builder
		sb.WriteString("*Details:*\n")
		for _, pair := range kv {
			sb.WriteString(fmt.Sprintf("• *%s:* %s\n", pair[0], pair[1]))
		}
		blocks = append(blocks, slackBlock{
			Type: "section",
			Text: &slackText{Type: "mrkdwn", Text: strings.TrimSpace(sb.String())},
		})
	}

	return slackPayload{
		Text:   fmt.Sprintf("%s — %s", title, summary),
		Blocks: blocks,
	}
}

// =============================================================================
// Discord
// =============================================================================

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
}

type discordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

func discordColorForSeverity(s eventSeverity) int {
	switch s {
	case severityError:
		return 0xE74C3C
	case severityWarning:
		return 0xF1C40F
	case severitySuccess:
		return 0x2ECC71
	default:
		return 0x3498DB
	}
}

func buildDiscordPayload(event Event) discordPayload {
	title := strings.TrimSpace(event.Type)
	if title == "" {
		title = "NTM Event"
	}

	fields := make([]discordEmbedField, 0, 8)
	if event.Session != "" {
		fields = append(fields, discordEmbedField{Name: "Session", Value: event.Session, Inline: true})
	}
	if event.Agent != "" {
		fields = append(fields, discordEmbedField{Name: "Agent", Value: event.Agent, Inline: true})
	}
	if event.Pane != "" {
		fields = append(fields, discordEmbedField{Name: "Pane", Value: event.Pane, Inline: true})
	}
	if event.ID != "" {
		fields = append(fields, discordEmbedField{Name: "Event ID", Value: event.ID, Inline: false})
	}

	for _, pair := range sortedKV(event.Details) {
		fields = append(fields, discordEmbedField{Name: pair[0], Value: pair[1], Inline: false})
	}

	embed := discordEmbed{
		Title:       title,
		Description: formatSummary(event),
		Timestamp:   formatRFC3339(event.Timestamp),
		Color:       discordColorForSeverity(classifySeverity(event.Type)),
		Fields:      fields,
	}

	return discordPayload{
		Content: "NTM notification",
		Embeds:  []discordEmbed{embed},
	}
}

// =============================================================================
// Microsoft Teams (Adaptive Card)
// =============================================================================

type teamsFact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type teamsFactSet struct {
	Type  string      `json:"type"`
	Facts []teamsFact `json:"facts"`
}

type teamsTextBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Wrap   bool   `json:"wrap,omitempty"`
	Size   string `json:"size,omitempty"`
	Weight string `json:"weight,omitempty"`
	Color  string `json:"color,omitempty"`
}

type adaptiveCard struct {
	Schema  string        `json:"$schema"`
	Type    string        `json:"type"`
	Version string        `json:"version"`
	Body    []interface{} `json:"body"`
}

type teamsAttachment struct {
	ContentType string       `json:"contentType"`
	Content     adaptiveCard `json:"content"`
}

type teamsPayload struct {
	Type        string            `json:"type"`
	Attachments []teamsAttachment `json:"attachments"`
}

func buildTeamsPayload(event Event) teamsPayload {
	sev := classifySeverity(event.Type)
	color := "Accent"
	switch sev {
	case severityError:
		color = "Attention"
	case severityWarning:
		color = "Warning"
	case severitySuccess:
		color = "Good"
	}

	title := strings.TrimSpace(event.Type)
	if title == "" {
		title = "NTM Event"
	}

	facts := make([]teamsFact, 0, 8)
	if event.Session != "" {
		facts = append(facts, teamsFact{Title: "Session", Value: event.Session})
	}
	if event.Agent != "" {
		facts = append(facts, teamsFact{Title: "Agent", Value: event.Agent})
	}
	if event.Pane != "" {
		facts = append(facts, teamsFact{Title: "Pane", Value: event.Pane})
	}
	if ts := formatRFC3339(event.Timestamp); ts != "" {
		facts = append(facts, teamsFact{Title: "Time", Value: ts})
	}
	for _, pair := range sortedKV(event.Details) {
		facts = append(facts, teamsFact{Title: pair[0], Value: pair[1]})
	}

	body := []interface{}{
		teamsTextBlock{
			Type:   "TextBlock",
			Text:   title,
			Weight: "Bolder",
			Size:   "Medium",
			Color:  color,
		},
		teamsTextBlock{
			Type: "TextBlock",
			Text: formatSummary(event),
			Wrap: true,
		},
	}
	if len(facts) > 0 {
		body = append(body, teamsFactSet{Type: "FactSet", Facts: facts})
	}

	card := adaptiveCard{
		Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
		Type:    "AdaptiveCard",
		Version: "1.5",
		Body:    body,
	}

	return teamsPayload{
		Type: "message",
		Attachments: []teamsAttachment{
			{ContentType: "application/vnd.microsoft.card.adaptive", Content: card},
		},
	}
}
