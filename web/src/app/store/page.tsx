"use client";

import { useState, useEffect, useCallback } from "react";
import { listAgents, subscribeToAgent, checkEntitlement, mintEntitlementToken, type AgentInfo, type EntitlementInfo } from "@/lib/api";

const CATEGORIES = ["all", "coding", "refactor", "testing", "docs", "security", "review"];

export default function StorePage() {
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("all");
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Map agent name -> entitlement info if subscribed
  const [subscriptions, setSubscriptions] = useState<Record<string, EntitlementInfo>>({});
  const [subscribing, setSubscribing] = useState<Record<string, boolean>>({});

  const loadAgents = useCallback(async () => {
    setLoading(true);
    setError(null);
    const result = await listAgents(search || undefined, category !== "all" ? category : undefined);
    if (result.error) {
      setError(result.error);
      // Fall back to empty
      setAgents([]);
    } else {
      setAgents(result.data ?? []);
    }
    setLoading(false);
  }, [search, category]);

  useEffect(() => {
    loadAgents();
  }, [loadAgents]);

  // After agents load, check entitlement status for hosted agents
  useEffect(() => {
    if (!agents.length) return;
    (async () => {
      const token = localStorage.getItem("token");
      if (!token) return; // not logged in
      const subMap: Record<string, EntitlementInfo> = {};
      for (const agent of agents) {
        if (!agent.is_hosted) continue;
        // Check if we have an active entitlement
        const result = await checkEntitlement(agent.name);
        if (result.data?.allowed && result.data.entitlement_id) {
          // Try to fetch the full entitlement info
          // For now just mark as subscribed if allowed
          subMap[agent.name] = {
            id: result.data.entitlement_id,
            agent_id: agent.name,
            status: "active",
            period_start: "",
            period_end: "",
            quota_calls: result.data.quota_calls,
            quota_tokens: result.data.quota_tokens,
            used_calls: result.data.used_calls,
            used_tokens: result.data.used_tokens,
          };
        }
      }
      setSubscriptions(subMap);
    })();
  }, [agents]);

  async function handleSubscribe(agent: AgentInfo) {
    const token = localStorage.getItem("token");
    if (!token) {
      alert("Please log in first to subscribe to agents.");
      return;
    }
    setSubscribing((prev) => ({ ...prev, [agent.name]: true }));
    try {
      const result = await subscribeToAgent(agent.name);
      if (result.error) {
        alert(`Failed to subscribe: ${result.error}`);
        return;
      }
      const ent = result.data!;
      setSubscriptions((prev) => ({ ...prev, [agent.name]: ent }));
      alert(`Subscribed! You now have access to ${agent.name}.`);
    } finally {
      setSubscribing((prev) => ({ ...prev, [agent.name]: false }));
    }
  }

  async function handleConnectMCP(agent: AgentInfo) {
    const sub = subscriptions[agent.name];
    if (!sub) {
      alert(`Subscribe to ${agent.name} first.`);
      return;
    }
    const token = localStorage.getItem("token");
    if (!token) return;

    // Mint a short-lived entitlement token for the MCP gateway
    const result = await mintEntitlementToken(sub.id);
    if (result.error || !result.data) {
      alert(`Failed to get MCP token: ${result.error}`);
      return;
    }

    const mcpToken = result.data.token;
    const gatewayUrl = process.env.NEXT_PUBLIC_MCP_GATEWAY_URL || "http://localhost:3001";
    const mcpCommand = `claude mcp add ${agent.name} -- agentx mcp --transport http --port 3001`;

    // Store the MCP token in localStorage for the CLI to pick up
    localStorage.setItem(`entitlement_token_${agent.name}`, mcpToken);
    localStorage.setItem(`entitlement_id_${agent.name}`, sub.id);

    alert(
      `Connect to ${agent.name} using:\n\n` +
        `1. Start the MCP gateway:\n   agentx mcp --transport http --port 3001\n\n` +
        `2. In Claude Code, connect:\n   ${mcpCommand}\n\n` +
        `Your entitlement token has been stored for this session.`
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <h1 className="text-3xl font-bold text-gray-900">Agent Store</h1>
      <p className="mt-2 text-gray-600">
        Browse hosted agents. Subscribe to call them from your coding agent.
      </p>

      {/* Filters */}
      <div className="mt-8 flex gap-4">
        <input
          type="text"
          placeholder="Search agents..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 rounded-md border border-gray-300 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
        />
        <select
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          className="rounded-md border border-gray-300 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
        >
          <option value="all">All Categories</option>
          {CATEGORIES.slice(1).map((cat) => (
            <option key={cat} value={cat}>
              {cat.charAt(0).toUpperCase() + cat.slice(1)}
            </option>
          ))}
        </select>
        <button
          onClick={loadAgents}
          className="rounded-md bg-gray-100 px-4 py-2 text-sm text-gray-700 hover:bg-gray-200"
        >
          Refresh
        </button>
      </div>

      {/* Loading / Error */}
      {loading && <p className="mt-8 text-gray-500">Loading agents...</p>}
      {error && !loading && <p className="mt-8 text-red-500">Error: {error}</p>}

      {/* Agent Grid */}
      {!loading && !error && (
        <div className="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {agents.map((agent) => {
            const sub = subscriptions[agent.name];
            return (
              <div
                key={agent.name}
                className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-shadow"
              >
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">
                      {agent.display_name || agent.name}
                    </h3>
                    <span className="text-sm text-gray-500">
                      v{agent.version}
                    </span>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800">
                      {agent.category || "general"}
                    </span>
                    {agent.is_hosted && (
                      <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                        hosted
                      </span>
                    )}
                  </div>
                </div>
                <p className="mt-4 text-sm text-gray-600">{agent.description}</p>

                {sub ? (
                  /* Subscribed — show quota */
                  <div className="mt-4 bg-green-50 rounded-md p-3">
                    <p className="text-xs font-medium text-green-800">
                      Subscribed
                    </p>
                    <p className="text-xs text-green-700 mt-1">
                      {sub.used_calls}/{sub.quota_calls} calls used
                    </p>
                    <div className="mt-2 w-full bg-green-200 rounded-full h-1.5">
                      <div
                        className="bg-green-600 h-1.5 rounded-full"
                        style={{ width: `${Math.min(100, (sub.used_calls / sub.quota_calls) * 100)}%` }}
                      />
                    </div>
                  </div>
                ) : agent.price_cents != null ? (
                  <p className="mt-4 text-sm font-semibold text-gray-900">
                    ${(agent.price_cents / 100).toFixed(2)}/mo
                  </p>
                ) : null}

                <div className="mt-6 flex items-center justify-between">
                  <span className="text-sm text-gray-500">
                    {agent.download_count?.toLocaleString() ?? 0} installs
                  </span>
                  {agent.is_hosted ? (
                    sub ? (
                      <button
                        onClick={() => handleConnectMCP(agent)}
                        className="rounded-md bg-green-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-green-700"
                      >
                        Connect MCP
                      </button>
                    ) : (
                      <button
                        onClick={() => handleSubscribe(agent)}
                        disabled={subscribing[agent.name]}
                        className="rounded-md bg-gray-900 px-3 py-1.5 text-sm font-semibold text-white hover:bg-gray-700 disabled:opacity-50"
                      >
                        {subscribing[agent.name] ? "Subscribing..." : "Subscribe"}
                      </button>
                    )
                  ) : (
                    <button className="rounded-md bg-gray-100 px-3 py-1.5 text-sm font-semibold text-gray-700 hover:bg-gray-200">
                      Install
                    </button>
                  )}
                </div>

                <div className="mt-3">
                  <code className="text-xs bg-gray-100 px-2 py-1 rounded text-gray-700">
                    {agent.is_hosted
                      ? `Subscribe to access via MCP`
                      : `agentx install ${agent.name}`}
                  </code>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {!loading && !error && agents.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500">No agents found matching your search.</p>
        </div>
      )}
    </div>
  );
}
