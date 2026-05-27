"use client";

import { useState } from "react";

interface Memory {
  id: string;
  kind: string;
  content: string;
  source: string;
  created_at: string;
}

const MOCK_MEMORIES: Memory[] = [
  {
    id: "1",
    kind: "decision",
    content: "Use pnpm as package manager for all projects",
    source: "user",
    created_at: "2025-01-15",
  },
  {
    id: "2",
    kind: "fact",
    content: "Database uses PostgreSQL with pgvector extension",
    source: "agent",
    created_at: "2025-01-14",
  },
  {
    id: "3",
    kind: "preference",
    content: "Prefer functional components over class components in React",
    source: "user",
    created_at: "2025-01-13",
  },
  {
    id: "4",
    kind: "context",
    content: "Project follows trunk-based development with feature flags",
    source: "user",
    created_at: "2025-01-12",
  },
  {
    id: "5",
    kind: "fact",
    content: "API rate limit is 100 requests per minute per user",
    source: "agent",
    created_at: "2025-01-11",
  },
];

export default function MemoryPage() {
  const [memories] = useState<Memory[]>(MOCK_MEMORIES);
  const [search, setSearch] = useState("");
  const [kindFilter, setKindFilter] = useState("all");

  const filtered = memories.filter((m) => {
    const matchesSearch =
      search === "" ||
      m.content.toLowerCase().includes(search.toLowerCase());
    const matchesKind = kindFilter === "all" || m.kind === kindFilter;
    return matchesSearch && matchesKind;
  });

  const kindColor = (kind: string) => {
    switch (kind) {
      case "decision":
        return "bg-blue-100 text-blue-800";
      case "fact":
        return "bg-green-100 text-green-800";
      case "preference":
        return "bg-purple-100 text-purple-800";
      case "context":
        return "bg-yellow-100 text-yellow-800";
      default:
        return "bg-gray-100 text-gray-800";
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Memory</h1>
          <p className="mt-2 text-gray-600">
            Project memories that persist across agent sessions.
          </p>
        </div>
        <button className="rounded-md bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-700">
          Add Memory
        </button>
      </div>

      {/* Filters */}
      <div className="mt-8 flex gap-4">
        <input
          type="text"
          placeholder="Search memories..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 rounded-md border border-gray-300 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
        />
        <select
          value={kindFilter}
          onChange={(e) => setKindFilter(e.target.value)}
          className="rounded-md border border-gray-300 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
        >
          <option value="all">All Types</option>
          <option value="decision">Decision</option>
          <option value="fact">Fact</option>
          <option value="preference">Preference</option>
          <option value="context">Context</option>
        </select>
      </div>

      {/* Memory List */}
      <div className="mt-8 space-y-4">
        {filtered.map((memory) => (
          <div
            key={memory.id}
            className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow"
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <p className="text-gray-900">{memory.content}</p>
                <div className="mt-2 flex items-center gap-3">
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${kindColor(
                      memory.kind
                    )}`}
                  >
                    {memory.kind}
                  </span>
                  <span className="text-sm text-gray-500">
                    {memory.source}
                  </span>
                  <span className="text-sm text-gray-500">
                    {memory.created_at}
                  </span>
                </div>
              </div>
              <button className="text-gray-400 hover:text-gray-600">
                <svg
                  className="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            </div>
          </div>
        ))}
      </div>

      {filtered.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500">No memories found.</p>
          <p className="mt-2 text-sm text-gray-400">
            Add memories with{" "}
            <code className="bg-gray-100 px-1 rounded">
              agentx memory add "..."
            </code>
          </p>
        </div>
      )}

      {/* CLI Integration */}
      <div className="mt-12 bg-gray-900 rounded-lg p-6">
        <h2 className="text-white font-semibold mb-4">
          CLI Integration
        </h2>
        <pre className="text-gray-300 text-sm">
{`# Add a memory
agentx memory add "Use pnpm as package manager" --kind decision

# Search memories
agentx memory search "package manager"

# Sync to cloud
agentx memory sync`}
        </pre>
      </div>
    </div>
  );
}
