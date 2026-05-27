"use client";

import { useState } from "react";

const MOCK_AGENTS = [
  {
    name: "code-reviewer",
    display_name: "Code Reviewer",
    description: "Automated code review with best practices",
    version: "0.1.0",
    category: "coding",
    download_count: 1234,
  },
  {
    name: "refactor-assistant",
    display_name: "Refactor Assistant",
    description: "Intelligent refactoring suggestions",
    version: "0.2.1",
    category: "refactor",
    download_count: 890,
  },
  {
    name: "test-writer",
    display_name: "Test Writer",
    description: "Generate comprehensive test suites",
    version: "0.1.2",
    category: "testing",
    download_count: 567,
  },
  {
    name: "doc-generator",
    display_name: "Doc Generator",
    description: "Generate documentation from code",
    version: "0.3.0",
    category: "docs",
    download_count: 432,
  },
];

export default function StorePage() {
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("all");

  const filtered = MOCK_AGENTS.filter((agent) => {
    const matchesSearch =
      search === "" ||
      agent.name.includes(search.toLowerCase()) ||
      agent.description.toLowerCase().includes(search.toLowerCase());
    const matchesCategory =
      category === "all" || agent.category === category;
    return matchesSearch && matchesCategory;
  });

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <h1 className="text-3xl font-bold text-gray-900">Agent Store</h1>
      <p className="mt-2 text-gray-600">
        Browse and install agent packs for your coding workflow.
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
          <option value="coding">Coding</option>
          <option value="refactor">Refactor</option>
          <option value="testing">Testing</option>
          <option value="docs">Docs</option>
        </select>
      </div>

      {/* Agent Grid */}
      <div className="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {filtered.map((agent) => (
          <div
            key={agent.name}
            className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-shadow"
          >
            <div className="flex items-start justify-between">
              <div>
                <h3 className="text-lg font-semibold text-gray-900">
                  {agent.display_name}
                </h3>
                <span className="text-sm text-gray-500">
                  v{agent.version}
                </span>
              </div>
              <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800">
                {agent.category}
              </span>
            </div>
            <p className="mt-4 text-sm text-gray-600">{agent.description}</p>
            <div className="mt-6 flex items-center justify-between">
              <span className="text-sm text-gray-500">
                {agent.download_count.toLocaleString()} installs
              </span>
              <button className="rounded-md bg-gray-900 px-3 py-1.5 text-sm font-semibold text-white hover:bg-gray-700">
                Install
              </button>
            </div>
            <div className="mt-4">
              <code className="text-xs bg-gray-100 px-2 py-1 rounded text-gray-700">
                agentx install {agent.name}
              </code>
            </div>
          </div>
        ))}
      </div>

      {filtered.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500">No agents found matching your search.</p>
        </div>
      )}
    </div>
  );
}
