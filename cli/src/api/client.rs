use anyhow::Result;
use reqwest::Client;
use crate::config::GlobalConfig;

pub struct ApiClient {
    client: Client,
    base_url: String,
    token: Option<String>,
}

impl ApiClient {
    pub fn new() -> Result<Self> {
        let config = GlobalConfig::load()?;
        Ok(Self {
            client: Client::new(),
            base_url: config.api_base_url().to_string(),
            token: config.token,
        })
    }

    pub fn with_token(mut self, token: String) -> Self {
        self.token = Some(token);
        self
    }

    fn url(&self, path: &str) -> String {
        format!("{}{}", self.base_url, path)
    }

    pub async fn get(&self, path: &str) -> Result<reqwest::Response> {
        let mut req = self.client.get(self.url(path));
        if let Some(token) = &self.token {
            req = req.bearer_auth(token);
        }
        let resp = req.send().await?;
        Ok(resp)
    }

    pub async fn post<T: serde::Serialize>(&self, path: &str, body: &T) -> Result<reqwest::Response> {
        let mut req = self.client.post(self.url(path)).json(body);
        if let Some(token) = &self.token {
            req = req.bearer_auth(token);
        }
        let resp = req.send().await?;
        Ok(resp)
    }

    pub async fn put<T: serde::Serialize>(&self, path: &str, body: &T) -> Result<reqwest::Response> {
        let mut req = self.client.put(self.url(path)).json(body);
        if let Some(token) = &self.token {
            req = req.bearer_auth(token);
        }
        let resp = req.send().await?;
        Ok(resp)
    }

    pub async fn delete(&self, path: &str) -> Result<reqwest::Response> {
        let mut req = self.client.delete(self.url(path));
        if let Some(token) = &self.token {
            req = req.bearer_auth(token);
        }
        let resp = req.send().await?;
        Ok(resp)
    }
}
