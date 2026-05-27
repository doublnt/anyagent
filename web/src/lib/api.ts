const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface ApiResponse<T> {
  data?: T;
  error?: string;
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const token =
    typeof window !== "undefined" ? localStorage.getItem("token") : null;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers,
    });

    if (!res.ok) {
      const err = await res.text();
      return { error: err || `HTTP ${res.status}` };
    }

    const data = await res.json();
    return { data };
  } catch (err: any) {
    return { error: err.message || "Network error" };
  }
}

// Auth
export async function login(email: string, password?: string) {
  return request<{ token: string; user_id: string; email: string }>(
    "/api/v1/auth/login",
    {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }
  );
}

export async function register(email: string, name: string, password?: string) {
  return request<{ token: string; user_id: string; email: string }>(
    "/api/v1/auth/register",
    {
      method: "POST",
      body: JSON.stringify({ email, name, password }),
    }
  );
}

export async function getCurrentUser() {
  return request<{ id: string; email: string; name: string; plan: string }>(
    "/api/v1/auth/me"
  );
}

// Agents
export interface AgentInfo {
  name: string;
  display_name: string;
  description: string;
  version: string;
  category: string;
  tags: string[];
  author: string;
  download_count: number;
}

export async function listAgents(query?: string, category?: string) {
  const params = new URLSearchParams();
  if (query) params.set("q", query);
  if (category) params.set("category", category);
  return request<AgentInfo[]>(`/api/v1/agents?${params}`);
}

export async function getAgent(name: string) {
  return request<AgentInfo>(`/api/v1/agents/${name}`);
}

// Projects
export interface Project {
  id: string;
  name: string;
  repo_url?: string;
  created_at: string;
}

export async function listProjects() {
  return request<Project[]>("/api/v1/projects");
}

export async function createProject(name: string, repoUrl?: string) {
  return request<Project>("/api/v1/projects", {
    method: "POST",
    body: JSON.stringify({ name, repo_url: repoUrl }),
  });
}

// Memory
export interface Memory {
  id: string;
  project_id: string;
  kind: string;
  content: string;
  source: string;
  created_at: string;
}

export async function listMemories(projectId: string) {
  return request<Memory[]>(`/api/v1/projects/${projectId}/memories`);
}

export async function createMemory(
  projectId: string,
  kind: string,
  content: string
) {
  return request<Memory>(`/api/v1/projects/${projectId}/memories`, {
    method: "POST",
    body: JSON.stringify({ kind, content }),
  });
}

export async function searchMemories(projectId: string, query: string) {
  return request<Memory[]>(`/api/v1/projects/${projectId}/memories/search`, {
    method: "POST",
    body: JSON.stringify({ query }),
  });
}

// Traces
export interface Trace {
  id: string;
  project_id: string;
  agent_name?: string;
  task_description: string;
  status: string;
  started_at: string;
  finished_at?: string;
}

export interface TraceSpan {
  id: string;
  trace_id: string;
  span_id: string;
  tool_name: string;
  input?: string;
  output?: string;
  status: string;
}

export async function listTraces(projectId: string) {
  return request<Trace[]>(`/api/v1/projects/${projectId}/traces`);
}

export async function getTrace(traceId: string) {
  return request<{ trace: Trace; spans: TraceSpan[] }>(
    `/api/v1/traces/${traceId}`
  );
}

// Subscription
export interface Subscription {
  plan: string;
  status: string;
  max_projects: number;
  max_traces_per_day: number;
  max_memory_mb: number;
}

export async function getSubscription() {
  return request<Subscription>("/api/v1/subscription");
}
