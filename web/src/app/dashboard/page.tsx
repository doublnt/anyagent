"use client";

import { useState, useEffect, useCallback } from "react";
import { listProjects, listAgents, getSubscription, type Project, type AgentInfo, type Subscription } from "@/lib/api";

export default function DashboardPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [subscription, setSubscription] = useState<Subscription | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    const token = localStorage.getItem("token");
    if (!token) {
      setLoading(false);
      return;
    }
    const [projRes, agentRes, subRes] = await Promise.all([
      listProjects(),
      listAgents(),
      getSubscription(),
    ]);
    if (projRes.data) setProjects(projRes.data);
    if (agentRes.data) setAgents(agentRes.data);
    if (subRes.data) setSubscription(subRes.data as Subscription);
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const totalMemories = 0; // TODO: sum from projects
  const totalTraces = 0;   // TODO: sum from projects
  const installedAgents = agents.filter((a) => !a.is_hosted).length;
  const subscribedAgents = agents.filter((a) => a.is_hosted).length;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
          <p className="mt-2 text-gray-600">Manage your projects and subscriptions.</p>
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
            {loading ? "—" : projects.length}
          </p>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <p className="text-sm text-gray-500">Installed Agents</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">
            {loading ? "—" : installedAgents}
          </p>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <p className="text-sm text-gray-500">Hosted Subscriptions</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">
            {loading ? "—" : subscribedAgents}
          </p>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <p className="text-sm text-gray-500">Plan</p>
          <p className="mt-2 text-xl font-bold text-gray-900">
            {subscription?.plan ?? "—"}
          </p>
        </div>
      </div>

      {/* Projects List */}
      <div className="mt-8">
        <h2 className="text-xl font-semibold text-gray-900">Your Projects</h2>
        {loading ? (
          <p className="mt-4 text-gray-500">Loading...</p>
        ) : projects.length === 0 ? (
          <div className="mt-4 bg-gray-50 rounded-lg p-8 text-center">
            <p className="text-gray-500">No projects yet.</p>
            <p className="text-sm text-gray-400 mt-1">
              Run <code>agentx init</code> in your project directory to get started.
            </p>
          </div>
        ) : (
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
                        {project.repo_url && (
                          <p className="text-sm text-gray-500">{project.repo_url}</p>
                        )}
                      </div>
                      <div className="text-gray-400">
                        <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                        </svg>
                      </div>
                    </div>
                  </a>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      {/* Quick Actions */}
      <div className="mt-8">
        <h2 className="text-xl font-semibold text-gray-900">Quick Start</h2>
        <div className="mt-4 bg-gray-900 rounded-lg p-6">
          <pre className="text-gray-300 text-sm">
{`# Initialize a project
cd your-project
agentx init

# Browse the agent store
open http://localhost:3000/store

# Start MCP server
agentx mcp

# Then in Claude Code:
claude mcp add agentx -- agentx mcp`}
          </pre>
        </div>
      </div>
    </div>
  );
}
