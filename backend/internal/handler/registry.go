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

	"gopkg.in/yaml.v3"

	"github.com/anyagent/anyagent/backend/internal/db"
	"github.com/anyagent/anyagent/backend/internal/middleware"
)

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
	IsHosted      bool     `json:"is_hosted"`
	PriceCents    *int     `json:"price_cents,omitempty"`
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

	agents, err := db.ListAgents(r.Context(), query, category, 50, 0)
	if err != nil {
		// Fall back to empty list if DB is unavailable
		agents = []db.Agent{}
	}

	out := make([]AgentInfo, 0, len(agents))
	for _, a := range agents {
		out = append(out, AgentInfo{
			Name:          a.Name,
			DisplayName:   a.DisplayName,
			Description:   a.Description,
			Version:       a.Version,
			Category:      a.Category,
			Tags:          a.Tags,
			Author:        a.AuthorID,
			DownloadCount: 0,
			IsHosted:      a.IsHosted,
			PriceCents:    a.PriceCents,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func GetAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	a, err := db.GetAgentByName(r.Context(), name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %s", name), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AgentInfo{
		Name:          a.Name,
		DisplayName:   a.DisplayName,
		Description:   a.Description,
		Version:       a.Version,
		Category:      a.Category,
		Tags:          a.Tags,
		Author:        a.AuthorID,
		IsHosted:      a.IsHosted,
		PriceCents:    a.PriceCents,
	})
}

func ListAgentVersions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	a, err := db.GetAgentByName(r.Context(), name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %s", name), http.StatusNotFound)
		return
	}

	versions, err := db.ListAgentVersions(r.Context(), a.ID)
	if err != nil {
		versions = []db.AgentVersion{}
	}

	out := make([]map[string]string, 0, len(versions))
	for _, v := range versions {
		out = append(out, map[string]string{
			"version":     v.Version,
			"artifact_url": v.ArtifactURL,
			"checksum":     v.Checksum,
			"created_at":   v.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func DownloadAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	version := r.PathValue("version")

	a, err := db.GetAgentByName(r.Context(), name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %s", name), http.StatusNotFound)
		return
	}

	var artifactURL string
	if version != "" {
		v, err := db.GetAgentVersion(r.Context(), a.ID, version)
		if err != nil {
			http.Error(w, fmt.Sprintf("Version not found: %s", version), http.StatusNotFound)
			return
		}
		artifactURL = v.ArtifactURL
	} else {
		versions, _ := db.ListAgentVersions(r.Context(), a.ID)
		if len(versions) > 0 {
			artifactURL = versions[0].ArtifactURL
		}
	}

	if artifactURL == "" {
		// Fallback: build tarball from filesystem
		serveFromFilesystem(w, r, name, version)
		return
	}

	// Stream from object store / URL (for MVP, redirect to local path)
	http.ServeFile(w, r, artifactURL)
}

func serveFromFilesystem(w http.ResponseWriter, r *http.Request, name, version string) {
	agentDir := filepath.Join(AgentStoreDir, name)
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Agent not found: %s", name), http.StatusNotFound)
		return
	}

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
		relPath, err := filepath.Rel(agentDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return err
		}
		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tw, file)
		return err
	})
	if err != nil {
		// headers already sent; just log
		fmt.Fprintf(os.Stderr, "tar error: %v\n", err)
	}
}

// PublishAgent handles agent publication.
// Requires scope=manage (set by RequireScope middleware on the route).
func PublishAgent(w http.ResponseWriter, r *http.Request) {
	authorID := middleware.GetUserID(r.Context())
	if authorID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	version := strings.TrimSpace(r.FormValue("version"))
	description := r.FormValue("description")
	category := r.FormValue("category")
	tagsRaw := r.FormValue("tags")
	isHostedStr := r.FormValue("is_hosted")
	priceCentsStr := r.FormValue("price_cents")

	if name == "" || version == "" {
		http.Error(w, `{"error":"name and version are required"}`, http.StatusBadRequest)
		return
	}

	var tags []string
	if tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ",") {
			if t := strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
	}

	isHosted := isHostedStr == "true" || isHostedStr == "1"

	var priceCents *int
	if priceCentsStr != "" {
		var pc int
		if _, err := fmt.Sscanf(priceCentsStr, "%d", &pc); err == nil {
			priceCents = &pc
		}
	}

	// Parse manifest from the pack
	var manifest AgentManifest
	if mfData := r.FormValue("manifest"); mfData != "" {
		if err := yaml.Unmarshal([]byte(mfData), &manifest); err != nil {
			http.Error(w, "Invalid manifest YAML: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	manifestJSON, _ := json.Marshal(manifest)

	// Upsert agent
	a, err := func() (*db.Agent, error) {
		existing, err := db.GetAgentByName(r.Context(), name)
		if err == nil && existing != nil {
			// Update existing — not implemented for MVP; just reuse
			return existing, nil
		}
		return db.CreateAgent(r.Context(), name, authorID, version,
			strings.TrimSpace(r.FormValue("display_name")),
			description, category, tags, manifestJSON, isHosted, priceCents)
	}()
	if err != nil {
		http.Error(w, "Failed to create agent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle tarball upload
	artifactURL := ""
	if file, _, err := r.FormFile("artifact"); err == nil {
		defer file.Close()

		// Store tarball in object store directory
		storeDir := filepath.Join(AgentStoreDir, name)
		if err := os.MkdirAll(storeDir, 0755); err != nil {
			http.Error(w, "Failed to create store directory", http.StatusInternalServerError)
			return
		}

		destPath := filepath.Join(storeDir, fmt.Sprintf("%s.tar.gz", version))
		dest, err := os.Create(destPath)
		if err != nil {
			http.Error(w, "Failed to create artifact file", http.StatusInternalServerError)
			return
		}
		defer dest.Close()

		if _, err := io.Copy(dest, file); err != nil {
			http.Error(w, "Failed to save artifact", http.StatusInternalServerError)
			return
		}

		artifactURL = destPath

		// Checksum
		checksum := ""
		if cs := r.FormValue("checksum"); cs != "" {
			checksum = cs
		}

		if _, err := db.CreateAgentVersion(r.Context(), a.ID, version, "", artifactURL, checksum); err != nil {
			// Non-fatal; log it
			fmt.Fprintf(os.Stderr, "Warning: failed to create agent version: %v\n", err)
		}

		// Extract pack to filesystem so DownloadAgent can serve it
		file.Seek(0, 0)
		if err := extractTarballHardened(file, storeDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to extract tarball: %v\n", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":         a.Name,
		"version":      version,
		"is_hosted":    a.IsHosted,
		"artifact_url": artifactURL,
		"status":       "published",
	})
}

// extractTarballHardened extracts a tar.gz to destDir with zip-slip and size protection.
func extractTarballHardened(reader io.Reader, destDir string) error {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var totalSize int64
	const maxSize = 100 << 20 // 100 MB cap

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		// Security: prevent zip-slip (ensure target is inside destDir)
		cleanTarget := filepath.Clean(target)
		cleanDest := filepath.Clean(destDir)
		if !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("tar entry %q attempts path traversal", header.Name)
		}

		// Size cap
		totalSize += header.Size
		if totalSize > maxSize {
			return fmt.Errorf("tar archive exceeds %d MB size limit", maxSize>>20)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			f.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// loadManifestFromYAML parses a real YAML agent manifest.
func loadManifestFromYAML(path string) (*AgentManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m AgentManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// --- legacy stubs kept for download fallback ---
type legacyAgentInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	Category    string
}

func loadLegacyManifest(path string) (*legacyAgentInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m legacyAgentInfo
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
			m.Name = value
		case "version":
			m.Version = value
		case "description":
			m.Description = value
		case "author":
			m.Author = value
		case "category":
			m.Category = value
		}
	}
	if m.Name == "" {
		m.Name = filepath.Base(path)
		m.Version = "0.0.1"
	}
	return &m, nil
}

// Stub for GetAgent in non-MVP path — kept for DownloadAgent fallback
var _ = time.Time{}
