"use client";

import { useState } from "react";

interface Trace {
  id: string;
  task: string;
  agent: string;
  status: string;
  started_at: string;
  duration: string;
}

const MOCK_TRACES: Trace[] = [
  {
    id: "abc12345",
    task: "Review authentication module",
    agent: "code-reviewer",
    status: "completed",
    started_at: "2025-01-15 10:30",
    duration: "2m 15s",
  },
  {
    id: "def67890",
    task: "Refactor database layer",
    agent: "refactor-assistant",
    status: "completed",
    started_at: "2025-01-15 09:15",
    duration: "5m 30s",
  },
  {
    id: "ghi11111",
    task: "Generate tests for user service",
    agent: "test-writer",
    status: "failed",
    started_at: "2025-01-15 08:00",
    duration: "1m 45s",
  },
  {
    id: "jkl22222",
    task: "Update API documentation",
    agent: "doc-generator",
    status: "running",
    started_at: "2025-01-15 11:00",
    duration: "...",
  },
];

export default function TracesPage() {
  const [traces] = useState<Trace[]>(MOCK_TRACES);
  const [selectedTrace, setSelectedTrace] = useState<Trace | null>(null);

  const statusColor = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-100 text-green-800";
      case "failed":
        return "bg-red-100 text-red-800";
      case "running":
        return "bg-yellow-100 text-yellow-800";
      default:
        return "bg-gray-100 text-gray-800";
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <h1 className="text-3xl font-bold text-gray-900">Traces</h1>
      <p className="mt-2 text-gray-600">
        View execution traces from your agent runs.
      </p>

      <div className="mt-8 flex gap-8">
        {/* Trace List */}
        <div className="flex-1">
          <div className="bg-white shadow overflow-hidden sm:rounded-md">
            <ul className="divide-y divide-gray-200">
              {traces.map((trace) => (
                <li key={trace.id}>
                  <button
                    onClick={() => setSelectedTrace(trace)}
                    className={`w-full text-left hover:bg-gray-50 px-4 py-4 sm:px-6 ${
                      selectedTrace?.id === trace.id ? "bg-gray-50" : ""
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-gray-900">
                          {trace.task}
                        </p>
                        <p className="text-sm text-gray-500">
                          {trace.agent} · {trace.started_at}
                        </p>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="text-sm text-gray-500">
                          {trace.duration}
                        </span>
                        <span
                          className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${statusColor(
                            trace.status
                          )}`}
                        >
                          {trace.status}
                        </span>
                      </div>
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Trace Detail */}
        <div className="w-96">
          {selectedTrace ? (
            <div className="bg-white shadow sm:rounded-lg p-6">
              <h3 className="text-lg font-medium text-gray-900">
                Trace Details
              </h3>
              <dl className="mt-4 space-y-4">
                <div>
                  <dt className="text-sm text-gray-500">ID</dt>
                  <dd className="text-sm font-mono text-gray-900">
                    {selectedTrace.id}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-gray-500">Task</dt>
                  <dd className="text-sm text-gray-900">
                    {selectedTrace.task}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-gray-500">Agent</dt>
                  <dd className="text-sm text-gray-900">
                    {selectedTrace.agent}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-gray-500">Status</dt>
                  <dd>
                    <span
                      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${statusColor(
                        selectedTrace.status
                      )}`}
                    >
                      {selectedTrace.status}
                    </span>
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-gray-500">Duration</dt>
                  <dd className="text-sm text-gray-900">
                    {selectedTrace.duration}
                  </dd>
                </div>
              </dl>

              <div className="mt-6">
                <h4 className="text-sm font-medium text-gray-900">Spans</h4>
                <div className="mt-2 space-y-2">
                  <div className="bg-gray-50 rounded p-3 text-sm">
                    <p className="font-medium">agentx_git_context</p>
                    <p className="text-gray-500 text-xs">0.5s</p>
                  </div>
                  <div className="bg-gray-50 rounded p-3 text-sm">
                    <p className="font-medium">agentx_read_file</p>
                    <p className="text-gray-500 text-xs">0.2s</p>
                  </div>
                  <div className="bg-gray-50 rounded p-3 text-sm">
                    <p className="font-medium">agentx_run_command</p>
                    <p className="text-gray-500 text-xs">1.8s</p>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="bg-white shadow sm:rounded-lg p-6 text-center text-gray-500">
              Select a trace to view details
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
