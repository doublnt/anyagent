package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// RunInput is the JSON sent to the sandbox container on stdin.
type RunInput struct {
	PackDir    string          `json:"pack_dir"`    // path to unpacked agent pack (RO mount)
	PromptText string          `json:"prompt_text"` // seller's system prompt
	Knowledge  []string        `json:"knowledge"`   // seller private files content
	UserInput  json.RawMessage `json:"user_input"`  // buyer's structured input (diff/files/task)
	ToolNames  []string        `json:"tool_names"`  // available tool names
	EgressURL  string          `json:"egress_url"`  // Harpoon egress proxy URL
	TokenCap   int             `json:"token_cap"`   // per-call token limit
	StepCap    int             `json:"step_cap"`     // per-call step limit
}

// RunOutput is the JSON received from the container on stdout.
type RunOutput struct {
	Result       string `json:"result"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	Error        string `json:"error,omitempty"`
}

// UsageSummary is returned alongside the output for metering.
type UsageSummary struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	CostMicros   int64 `json:"cost_micros"` // platform cost in micro-dollars
}

// Runner executes a hosted agent in an isolated gVisor container.
type Runner struct {
	BaseImage   string // e.g. "anyagent/agent-runner:latest"
	EgressProxy string // URL of the Harpoon egress proxy
	ModelAPIKey string // API key for the model provider (platform-owned)
	ModelURL    string // e.g. "https://api.anthropic.com/v1/messages"
	ModelName   string // e.g. "claude-3-5-sonnet-20241022"
}

const (
	defaultTokenCap = 8000  // per-call token budget
	defaultStepCap  = 20    // max LLM round-trips per call
	maxContentSize  = 10 << 20
)

// Run executes the agent loop inside an ephemeral gVisor container.
// It returns the result text, usage summary, and any error.
func (r *Runner) Run(ctx context.Context, packDir, promptText string, knowledge []string, userInput json.RawMessage, toolNames []string) (string, *UsageSummary, error) {
	input := RunInput{
		PackDir:    packDir,
		PromptText: promptText,
		Knowledge:  knowledge,
		UserInput:  userInput,
		ToolNames:  toolNames,
		EgressURL:  r.EgressProxy,
		TokenCap:   defaultTokenCap,
		StepCap:    defaultStepCap,
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return "", nil, fmt.Errorf("marshal input: %w", err)
	}

	if len(payload) > maxContentSize {
		return "", nil, fmt.Errorf("input exceeds maximum size")
	}

	// Create an ephemeral working directory for this run
	runID := uuid.New().String()
	runDir := filepath.Join(os.TempDir(), "anyagent-sandbox", runID)
	if err := os.MkdirAll(runDir, 0700); err != nil {
		return "", nil, fmt.Errorf("create run dir: %w", err)
	}
	defer os.RemoveAll(runDir)

	// Write input to stdin file
	stdinPath := filepath.Join(runDir, "input.json")
	if err := os.WriteFile(stdinPath, payload, 0600); err != nil {
		return "", nil, fmt.Errorf("write stdin: %w", err)
	}

	// The actual container execution uses runsc (gVisor).
	// For MVP on a machine with runsc installed:
	runArgs := []string{
		"run",
		"--network=none",       // no network to host
		"--cpus=0.5",           // limit CPU
		"--memory=256M",        // limit RAM
		"--platform=ptrace",     // gVisor platform
		"--root",
		r.BaseImage,
		"/runner",
		"--stdin", stdinPath,
	}

	cmd := exec.CommandContext(ctx, "runsc", runArgs...)
	cmd.Stdout = new(bytes.Buffer)
	cmd.Stderr = new(bytes.Buffer)

	timeout := 60 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout < 0 {
			timeout = 30 * time.Second
		}
	}
	cmd.WaitDelay = timeout // runsc will be killed after this

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start)

	if err != nil {
		// Container failed — return error but log
		return "", &UsageSummary{}, fmt.Errorf("sandbox run failed after %s: %w\nstderr: %s",
			elapsed, err, cmd.Stderr.(*bytes.Buffer).String())
	}

	var output RunOutput
	if err := json.Unmarshal(cmd.Stdout.(*bytes.Buffer).Bytes(), &output); err != nil {
		return "", &UsageSummary{}, fmt.Errorf("parse container output: %w", err)
	}

	if output.Error != "" {
		return "", &UsageSummary{}, fmt.Errorf("sandbox error: %s", output.Error)
	}

	usage := &UsageSummary{
		InputTokens:  output.InputTokens,
		OutputTokens: output.OutputTokens,
		CostMicros:   calculateCostMicros(output.InputTokens, output.OutputTokens),
	}

	return output.Result, usage, nil
}

// calculateCostMicros computes platform cost in micro-dollars.
func calculateCostMicros(inputTokens, outputTokens int64) int64 {
	inputMicros := inputTokens * 3    // $3/M = 3 microcents/token
	outputMicros := outputTokens * 15 // $15/M = 15 microcents/token
	margin := float64(inputMicros + outputMicros) * 0.30
	return inputMicros + outputMicros + int64(margin)
}

// CheckHealth verifies the sandbox runtime is available.
func (r *Runner) CheckHealth(ctx context.Context) error {
	if _, err := exec.LookPath("runsc"); err != nil {
		return fmt.Errorf("runsc not found in PATH: %w", err)
	}
	// Try a minimal runsc validation with 5s timeout
	ctx5, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx5, "runsc", "list")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("runsc list failed: %w", err)
	}
	return nil
}

// fetchPackArtifact downloads or copies the agent pack tarball to a local temp dir.
func FetchPackArtifact(ctx context.Context, artifactURL string) (string, error) {
	// MVP: artifactURL is a local path (./data/agents/<name>/<version>.tar.gz)
	if filepath.IsAbs(artifactURL) {
		return artifactURL, nil
	}
	resp, err := http.Get(artifactURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	runID := uuid.New().String()
	dest := filepath.Join(os.TempDir(), "anyagent-sandbox", runID, "pack.tar.gz")
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return "", err
	}
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return dest, nil
}
