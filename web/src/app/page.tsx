export default function Home() {
  return (
    <div className="bg-white">
      {/* Hero */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24">
        <div className="text-center">
          <h1 className="text-4xl font-bold tracking-tight text-gray-900 sm:text-6xl">
            Enhance Your Coding Agent
          </h1>
          <p className="mt-6 text-lg leading-8 text-gray-600 max-w-2xl mx-auto">
            Continue using Claude Code / Codex / Cursor while gaining Agent Packs,
            project memory, traces, and team collaboration.
          </p>
          <div className="mt-10 flex items-center justify-center gap-x-6">
            <a
              href="/store"
              className="rounded-md bg-gray-900 px-3.5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-gray-700"
            >
              Browse Agents
            </a>
            <a
              href="/docs"
              className="text-sm font-semibold leading-6 text-gray-900"
            >
              Read the docs <span aria-hidden="true">→</span>
            </a>
          </div>
        </div>

        {/* Quick Start */}
        <div className="mt-20 bg-gray-900 rounded-2xl p-8 max-w-3xl mx-auto">
          <h2 className="text-white text-lg font-semibold mb-4">Quick Start</h2>
          <pre className="text-gray-300 text-sm overflow-x-auto">
{`# Install CLI
cargo install agentx

# Initialize project
cd your-project
agentx init

# Install an agent
agentx install code-reviewer

# Start MCP server
agentx mcp

# Connect with Claude Code
claude mcp add agentx -- agentx mcp`}
          </pre>
        </div>
      </div>

      {/* Features */}
      <div className="bg-gray-50 py-24">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold text-gray-900">
              Why AnyAgent?
            </h2>
            <p className="mt-4 text-lg text-gray-600">
              Not an API relay. Not a prompt marketplace. A real agent enhancement layer.
            </p>
          </div>

          <div className="mt-16 grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3">
            <div className="bg-white p-6 rounded-lg shadow">
              <h3 className="text-lg font-semibold text-gray-900">Agent Packs</h3>
              <p className="mt-2 text-gray-600">
                Pre-configured agents with prompts, tools, and eval rules.
                Install with one command.
              </p>
            </div>
            <div className="bg-white p-6 rounded-lg shadow">
              <h3 className="text-lg font-semibold text-gray-900">Project Memory</h3>
              <p className="mt-2 text-gray-600">
                Persistent context across sessions. Your agent remembers
                decisions, preferences, and facts.
              </p>
            </div>
            <div className="bg-white p-6 rounded-lg shadow">
              <h3 className="text-lg font-semibold text-gray-900">Traces</h3>
              <p className="mt-2 text-gray-600">
                Full execution traces with tool calls, inputs, outputs,
                and timing. Debug with confidence.
              </p>
            </div>
            <div className="bg-white p-6 rounded-lg shadow">
              <h3 className="text-lg font-semibold text-gray-900">MCP Integration</h3>
              <p className="mt-2 text-gray-600">
                Works with Claude Code, Codex, Cursor via MCP protocol.
                No lock-in to any specific tool.
              </p>
            </div>
            <div className="bg-white p-6 rounded-lg shadow">
              <h3 className="text-lg font-semibold text-gray-900">Eval Rules</h3>
              <p className="mt-2 text-gray-600">
                Define and run evaluation rules on agent outputs.
                Ensure quality and consistency.
              </p>
            </div>
            <div className="bg-white p-6 rounded-lg shadow">
              <h3 className="text-lg font-semibold text-gray-900">Team Workspace</h3>
              <p className="mt-2 text-gray-600">
                Share agents, memories, and traces with your team.
                Collaborate on agent configurations.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
