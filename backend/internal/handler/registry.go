package handler

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentStoreDir is where agent packs are stored (configurable)
var AgentStoreDir = getEnv("AGENT_STORE_DIR", "./data/agents")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type AgentInfo struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name,omitempty"`
	Description   string   `json:"description,omitempty"`
	Version       string   `json:"version"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Author        string   `json:"author,omitempty"`
	DownloadCount int      `json:"download_count"`
}

type AgentManifest struct {
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description,omitempty"`
	Author      string       `json:"author,omitempty"`
	Category    string       `json:"category,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Prompts     []PromptFile `json:"prompts,omitempty"`
	Tools       []ToolFile   `json:"tools,omitempty"`
	Eval        []EvalFile   `json:"eval,omitempty"`
}

type PromptFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type ToolFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

type EvalFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func ListAgents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")

	entries, err := os.ReadDir(AgentStoreDir)
	if err != nil {
		// No agents directory yet
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]AgentInfo{})
		return
	}

	var agents []AgentInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(AgentStoreDir, entry.Name(), "agent.yaml")
		manifest, err := loadManifest(manifestPath)
		if err != nil {
			continue
		}

		tags := manifest.Tags
		if tags == nil {
			tags = []string{}
		}

		agent := AgentInfo{
			Name:        manifest.Name,
			DisplayName: manifest.Name,
			Description: manifest.Description,
			Version:     manifest.Version,
			Category:    manifest.Category,
			Tags:        tags,
			Author:      manifest.Author,
		}

		// Apply filters
		if query != "" && !strings.Contains(strings.ToLower(agent.Name), strings.ToLower(query)) &&
			!strings.Contains(strings.ToLower(agent.Description), strings.ToLower(query)) {
			continue
		}
		if category != "" && agent.Category != category {
			continue
		}

		agents = append(agents, agent)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func GetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	manifestPath := filepath.Join(AgentStoreDir, name, "agent.yaml")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %s", name), http.StatusNotFound)
		return
	}

	agent := AgentInfo{
		Name:        manifest.Name,
		DisplayName: manifest.Name,
		Description: manifest.Description,
		Version:     manifest.Version,
		Category:    manifest.Category,
		Tags:        manifest.Tags,
		Author:      manifest.Author,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func ListAgentVersions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	agentDir := filepath.Join(AgentStoreDir, name)
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Agent not found: %s", name), http.StatusNotFound)
		return
	}

	// For MVP, only one version per agent
	manifestPath := filepath.Join(agentDir, "agent.yaml")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		http.Error(w, "Failed to load agent", http.StatusInternalServerError)
		return
	}

	versions := []map[string]string{
		{"version": manifest.Version, "created_at": time.Now().Format(time.RFC3339)},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

func DownloadAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	version := r.PathValue("version")

	agentDir := filepath.Join(AgentStoreDir, name)
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Agent not found: %s", name), http.StatusNotFound)
		return
	}

	// Create tarball
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.tar.gz", name, version))

	gz := gzip.NewWriter(w)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	err := filepath.Walk(agentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(agentDir, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Skip directories (no content to write)
		if info.IsDir() {
			return nil
		}

		// Write file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})

	if err != nil {
		http.Error(w, "Failed to create tarball", http.StatusInternalServerError)
		return
	}
}

func PublishAgent(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	version := r.FormValue("version")
	description := r.FormValue("description")

	if name == "" || version == "" {
		http.Error(w, "name and version are required", http.StatusBadRequest)
		return
	}

	// Get uploaded tarball
	file, _, err := r.FormFile("artifact")
	if err != nil {
		http.Error(w, "artifact file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create agent directory
	agentDir := filepath.Join(AgentStoreDir, name)
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		http.Error(w, "Failed to create agent directory", http.StatusInternalServerError)
		return
	}

	// Extract tarball
	if err := extractTarball(file, agentDir); err != nil {
		http.Error(w, "Failed to extract tarball", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"name":        name,
		"version":     version,
		"description": description,
		"status":      "published",
	})
}

func loadManifest(path string) (*AgentManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest AgentManifest
	// Try YAML first (using simple key-value parsing for MVP)
	// In production, use a proper YAML library
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			manifest.Name = value
		case "version":
			manifest.Version = value
		case "description":
			manifest.Description = value
		case "author":
			manifest.Author = value
		case "category":
			manifest.Category = value
		}
	}

	// If parsing failed, return basic info from directory name
	if manifest.Name == "" {
		manifest.Name = filepath.Base(path)
		manifest.Version = "0.0.1"
	}

	return &manifest, nil
}

func extractTarball(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}

	return nil
}
