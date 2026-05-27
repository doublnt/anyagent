use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Trace {
    pub id: String,
    pub agent_name: Option<String>,
    pub task_description: String,
    pub status: String,
    pub started_at: String,
    pub finished_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TraceSpan {
    pub span_id: String,
    pub tool_name: String,
    pub input: Option<String>,
    pub output: Option<String>,
    pub status: String,
    pub duration_ms: Option<u64>,
}

#[derive(Debug, Serialize)]
pub struct CreateTraceRequest {
    pub task: String,
    pub agent_name: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct CreateSpanRequest {
    pub tool_name: String,
    pub input: Option<String>,
    pub output: Option<String>,
    pub status: String,
}
