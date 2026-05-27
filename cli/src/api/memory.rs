use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Memory {
    pub id: String,
    pub kind: String,
    pub content: String,
    pub source: Option<String>,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct CreateMemoryRequest {
    pub kind: String,
    pub content: String,
}

#[derive(Debug, Serialize)]
pub struct SearchMemoryRequest {
    pub query: String,
    pub limit: Option<u32>,
}
