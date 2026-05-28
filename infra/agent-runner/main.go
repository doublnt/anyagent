// Agent Runner — the loop driver that runs inside each sandboxed gVisor container.
// Reads RunInput from stdin, calls the LLM via the Harpoon egress proxy,
// executes declared tools (read_file, list_files, run_command) against the buyer
// input, and writes RunOutput to stdout.
package main

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
	"strings"
	"time"
)

// RunInput is received on stdin from the platform orchestrator.
type RunInput struct {
	PackDir    string   `json:"pack_dir"`    // path to seller pack (RO mount)
	PromptText string   `json:"prompt_text"` // seller's system prompt
	Knowledge  []string `json:"knowledge"`  // seller private knowledge content
	UserInput  string   `json:"user_input"` // buyer's structured input JSON
	EgressURL  string   `json:"egress_url"`  // Harpoon egress proxy URL
	TokenCap   int      `json:"token_cap"`  // max tokens per call
	StepCap    int      `json:"step_cap"`   // max LLM round-trips
	ModelName  string   `json:"model_name"` // e.g. claude-3-5-sonnet-20241022
}

// RunOutput is written to stdout.
type RunOutput struct {
	Result       string `json:"result"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	Error        string `json:"error,omitempty"`
}

func main() {
	// Read input from stdin
	var input RunInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		writeOutput(RunOutput{Error: fmt.Sprintf("decode input: %v", err)})
		os.Exit(1)
	}

	if input.TokenCap <= 0 {
		input.TokenCap = 8000
	}
	if input.StepCap <= 0 {
		input.StepCap = 20
	}
	if input.ModelName == "" {
		input.ModelName = "claude-3-5-sonnet-20241022"
	}

	result, inputTokens, outputTokens, err := runLoop(input)
	if err != nil {
		writeOutput(RunOutput{
			Result:       result,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Error:        err.Error(),
		})
		os.Exit(1)
	}

	writeOutput(RunOutput{
		Result:       result,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	})
}

func runLoop(input RunInput) (result string, inputTokens, outputTokens int64, err error) {
	// Assemble the system prompt from seller pack + knowledge
	systemPrompt := input.PromptText
	if len(input.Knowledge) > 0 {
		systemPrompt += "\n\n--- Seller Private Knowledge ---\n" +
			strings.Join(input.Knowledge, "\n\n")
	}

	// Build initial messages
	var messages []llmMessage
	messages = append(messages, llmMessage{
		Role:    "system",
		Content: systemPrompt,
	})
	messages = append(messages, llmMessage{
		Role:    "user",
		Content: input.UserInput,
	})

	var totalInput, totalOutput int64

	for step := 0; step < input.StepCap; step++ {
		resp, inp, out, callErr := callLLM(input.EgressURL, input.ModelName, messages)
		if callErr != nil {
			return "", totalInput, totalOutput, fmt.Errorf("llm call: %w", callErr)
		}
		totalInput += inp
		totalOutput += out

		if resp.StopReason == "end_turn" || resp.Content == nil {
			break
		}

		// If the model requested a tool call, execute it and continue
		if len(resp.Content) > 0 && resp.Content[0].Type == "tool_use" {
			toolResult := executeTool(resp.Content[0].Name, resp.Content[0].Input)
			messages = append(messages, llmMessage{
				Role:    "user",
				Content: "",
				ToolResults: []toolResult{{
					ToolUseID: resp.Content[0].ID,
					Content:   toolResult,
				}},
			})
			continue
		}

		// Otherwise return the text result
		return textContent(resp), totalInput, totalOutput, nil
	}

	return "(max steps reached, no result)", totalInput, totalOutput, nil
}

type llmMessage struct {
	Role         string          `json:"role"`
	Content      string          `json:"content,omitempty"`
	ToolResults  []toolResult    `json:"tool_results,omitempty"`
}

type toolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

type llmRequest struct {
	Model       string        `json:"model"`
MaxTokens   int           `json:"max_tokens"`
SystemPrompt string        `json:"system,omitempty"`
Messages    []llmMessage   `json:"messages"`
	Tools      []toolDef     `json:"tools,omitempty"`
}

type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema string `json:"input_schema"` // JSON schema as string
}

type llmResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      llmUsage       `json:"usage"`
}

type contentBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Name   string `json:"name,omitempty"`
	ID     string `json:"id,omitempty"`
	Input  map[string]interface{} `json:"input,omitempty"`
}

type llmUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

func callLLM(egressURL, model string, messages []llmMessage) (*llmResponse, int64, int64, error) {
	reqBody := llmRequest{
		Model:     model,
		MaxTokens: 4096,
		Messages:  messages,
		Tools: []toolDef{
			{
				Name:        "read_file",
				Description: "Read the contents of a file from the project",
				InputSchema: `{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`,
			},
			{
				Name:        "list_files",
				Description: "List files in a directory",
				InputSchema: `{"type":"object","properties":{"pattern":{"type":"string"}}}`,
			},
			{
				Name:        "run_command",
				Description: "Run a shell command",
				InputSchema: `{"type":"object","properties":{"command":{"type":"string"},"timeout_ms":{"type":"integer"}},"required":["command"]}`,
			},
		},
	}

	payload, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequest("POST", egressURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", os.Getenv("MODEL_API_KEY"))
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("egress request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, 0, 0, fmt.Errorf("egress returned %d: %s", resp.StatusCode, string(body))
	}

	var llmResp llmResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return nil, 0, 0, fmt.Errorf("decode llm response: %w", err)
	}

	return &llmResp, llmResp.Usage.InputTokens, llmResp.Usage.OutputTokens, nil
}

func executeTool(name, args map[string]interface{}) string {
	switch name {
	case "read_file":
		path, _ := args["path"].(string)
		if path == "" {
			return `{"error":"path is required"}`
		}
		// Path is relative to pack dir or buyer input dir
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Sprintf(`{"error":"%v"}`, err)
		}
		return string(data)

	case "list_files":
		pattern, _ := args["pattern"].(string)
		if pattern == "" {
			pattern = "*"
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Sprintf(`{"error":"%v"}`, err)
		}
		return strings.Join(matches, "\n")

	case "run_command":
		cmd, _ := args["command"].(string)
		if cmd == "" {
			return `{"error":"command is required"}`
		}
		timeout := 30000
		if t, ok := args["timeout_ms"].(float64); ok {
			timeout = int(t)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		defer cancel()
		parts := strings.SplitN(cmd, " ", 2)
		var c *exec.Cmd
		if len(parts) > 1 {
			c = exec.CommandContext(ctx, parts[0], parts[1])
		} else {
			c = exec.CommandContext(ctx, "sh", "-c", cmd)
		}
		c.Env = append(os.Environ(), "HOME=/tmp")
		out, err := c.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			return `{"error":"command timed out"}`
		}
		if err != nil {
			return fmt.Sprintf(`{"error":"%v","output":%s}`, err, string(out))
		}
		return string(out)

	default:
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, name)
	}
}

func textContent(r *llmResponse) string {
	var sb strings.Builder
	for _, block := range r.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String()
}

func writeOutput(o RunOutput) {
	json.NewEncoder(os.Stdout).Encode(o)
}
