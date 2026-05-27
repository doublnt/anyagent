package db

import (
	"context"
	"time"
)

type Trace struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	UserID          string     `json:"user_id"`
	AgentName       *string    `json:"agent_name,omitempty"`
	TaskDescription string     `json:"task_description"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

type TraceSpan struct {
	ID           string  `json:"id"`
	TraceID      string  `json:"trace_id"`
	SpanID       string  `json:"span_id"`
	ToolName     string  `json:"tool_name"`
	Input        *string `json:"input,omitempty"`
	Output       *string `json:"output,omitempty"`
	DurationMs   *int    `json:"duration_ms,omitempty"`
	Status       string  `json:"status"`
}

func ListTraces(ctx context.Context, projectID string, limit, offset int) ([]Trace, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id::text, project_id::text, user_id::text, agent_name, task_description, status, started_at, finished_at
		 FROM traces WHERE project_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`,
		projectID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var t Trace
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.UserID, &t.AgentName, &t.TaskDescription, &t.Status, &t.StartedAt, &t.FinishedAt); err != nil {
			return nil, err
		}
		traces = append(traces, t)
	}
	return traces, nil
}

func CreateTrace(ctx context.Context, projectID, userID, task string, agentName *string) (*Trace, error) {
	var t Trace
	err := Pool.QueryRow(ctx,
		`INSERT INTO traces (project_id, user_id, task_description, agent_name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, project_id::text, user_id::text, agent_name, task_description, status, started_at, finished_at`,
		projectID, userID, task, agentName,
	).Scan(&t.ID, &t.ProjectID, &t.UserID, &t.AgentName, &t.TaskDescription, &t.Status, &t.StartedAt, &t.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	var t Trace
	err := Pool.QueryRow(ctx,
		`SELECT id::text, project_id::text, user_id::text, agent_name, task_description, status, started_at, finished_at
		 FROM traces WHERE id = $1`,
		traceID,
	).Scan(&t.ID, &t.ProjectID, &t.UserID, &t.AgentName, &t.TaskDescription, &t.Status, &t.StartedAt, &t.FinishedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CompleteTrace(ctx context.Context, traceID, status string) error {
	_, err := Pool.Exec(ctx,
		`UPDATE traces SET status = $1, finished_at = NOW() WHERE id = $2`,
		status, traceID,
	)
	return err
}

func AddTraceSpan(ctx context.Context, traceID, spanID, toolName, status string, input, output *string, durationMs *int) (*TraceSpan, error) {
	var s TraceSpan
	err := Pool.QueryRow(ctx,
		`INSERT INTO trace_spans (trace_id, span_id, tool_name, input, output, duration_ms, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id::text, trace_id::text, span_id, tool_name, input, output, duration_ms, status`,
		traceID, spanID, toolName, input, output, durationMs, status,
	).Scan(&s.ID, &s.TraceID, &s.SpanID, &s.ToolName, &s.Input, &s.Output, &s.DurationMs, &s.Status)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetTraceSpans(ctx context.Context, traceID string) ([]TraceSpan, error) {
	rows, err := Pool.Query(ctx,
		`SELECT id::text, trace_id::text, span_id, tool_name, input, output, duration_ms, status
		 FROM trace_spans WHERE trace_id = $1 ORDER BY created_at`,
		traceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []TraceSpan
	for rows.Next() {
		var s TraceSpan
		if err := rows.Scan(&s.ID, &s.TraceID, &s.SpanID, &s.ToolName, &s.Input, &s.Output, &s.DurationMs, &s.Status); err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}
	return spans, nil
}
