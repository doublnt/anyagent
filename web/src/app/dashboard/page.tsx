"use client";

import { useState, useEffect } from "react";

interface Project {
  id: string;
  name: string;
  agents: number;
  memories: number;
  traces: number;
}

const MOCK_PROJECTS: Project[] = [
  { id: "1", name: "my-web-app", agents: 3, memories: 12, traces: 45 },
  { id: "2", name: "api-service", agents: 2, memories: 8, traces: 23 },
  { id: "3", name: "mobile-app", agents: 1, memories: 5, traces: 11 },
];

export default function DashboardPage() {
  const [projects, setProjects] = useState<Project[]>(MOCK_PROJECTS);

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
          <p className="mt-2 text-gray-600">
            Manage your projects and view activity.
          </p>
        </div>
        <button className="rounded-md bg-gray-900 px-4 py-2 text-sm font-semibold text-white hover:bg-gray-700">
          New Project
        </button>
      </div>

      {/* Stats */}
      <div className="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-4">
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <p className="text-sm text-gray-500">Projects</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">
            {projects.length}
          </p>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <p className="text-sm text-gray-500">Total Agents</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">
            {projects.reduce((sum, p) => sum + p.agents, 0)}
          </p>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <p className="text-sm text-gray-500">Memories</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">
            {projects.reduce((sum, p) => sum + p.memories, 0)}
          </p>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <p className="text-sm text-gray-500">Traces</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">
            {projects.reduce((sum, p) => sum + p.traces, 0)}
          </p>
        </div>
      </div>

      {/* Projects List */}
      <div className="mt-8">
        <h2 className="text-xl font-semibold text-gray-900">Your Projects</h2>
        <div className="mt-4 bg-white shadow overflow-hidden sm:rounded-md">
          <ul className="divide-y divide-gray-200">
            {projects.map((project) => (
              <li key={project.id}>
                <a
                  href={`/dashboard/projects/${project.id}`}
                  className="block hover:bg-gray-50 px-4 py-4 sm:px-6"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {project.name}
                      </p>
                      <p className="text-sm text-gray-500">
                        {project.agents} agents · {project.memories} memories ·{" "}
                        {project.traces} traces
                      </p>
                    </div>
                    <div className="text-gray-400">
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
                          d="M9 5l7 7-7 7"
                        />
                      </svg>
                    </div>
                  </div>
                </a>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="mt-8">
        <h2 className="text-xl font-semibold text-gray-900">Quick Start</h2>
        <div className="mt-4 bg-gray-900 rounded-lg p-6">
          <pre className="text-gray-300 text-sm">
{`# In your project directory
agentx init
agentx install code-reviewer
agentx mcp

# Then in Claude Code, the tools are available!`}
          </pre>
        </div>
      </div>
    </div>
  );
}
