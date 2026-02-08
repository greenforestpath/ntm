"use client";

/**
 * Sessions Dashboard
 *
 * Displays list of tmux sessions with their status and agents.
 */

import Link from "next/link";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getClient } from "@/lib/api/client";

export default function SessionsPage() {
  const [filter, setFilter] = useState("");
  const {
    data: sessions,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["sessions"],
    queryFn: async () => {
      const client = getClient();
      const response = await client.GET("/api/v1/sessions");
      if (response.error) {
        throw new Error(`Failed to fetch sessions: ${response.error}`);
      }
      return response.data;
    },
    refetchInterval: 10000, // Poll every 10 seconds as backup
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
        <p className="text-red-700 dark:text-red-400">
          Error loading sessions: {error.message}
        </p>
      </div>
    );
  }

  const sessionList = sessions?.sessions || [];
  const filteredSessions = useMemo(() => {
    if (!filter) return sessionList;
    const query = filter.toLowerCase();
    return sessionList.filter((session: Record<string, unknown>) => {
      const name = (session.name as string) || "";
      const tags = (session.tags as string[]) || [];
      return (
        name.toLowerCase().includes(query) ||
        tags.some((tag) => tag.toLowerCase().includes(query))
      );
    });
  }, [filter, sessionList]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
          Sessions
        </h1>
        <span className="text-sm text-gray-500 dark:text-gray-400">
          {filteredSessions.length} session
          {filteredSessions.length !== 1 ? "s" : ""}
        </span>
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="text-sm text-gray-500 dark:text-gray-400">
          Filter by session name or tags.
        </div>
        <input
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
          placeholder="Search sessions..."
          className="w-full sm:w-64 rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-sm text-gray-900 dark:text-white shadow-sm focus:border-blue-500 focus:outline-none"
        />
      </div>

      {filteredSessions.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-gray-500 dark:text-gray-400">
            No sessions found. Create one with{" "}
            <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">
              ntm spawn
            </code>
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredSessions.map((session: Record<string, unknown>) => (
            <SessionCard key={session.name as string} session={session} />
          ))}
        </div>
      )}
    </div>
  );
}

function SessionCard({ session }: { session: Record<string, unknown> }) {
  const name = session.name as string;
  const panes = (session.panes as unknown[]) || [];
  const tags = (session.tags as string[]) || [];
  const created = session.created_at as string;
  const sessionName = session.name as string;

  // Count agents by type
  const agentCounts = panes.reduce<Record<string, number>>((acc, pane) => {
    const agent = (pane as Record<string, unknown>).agent_type as string;
    if (agent) {
      acc[agent] = (acc[agent] || 0) + 1;
    }
    return acc;
  }, {});

  return (
    <Link
      href={`/sessions/${encodeURIComponent(sessionName)}`}
      className="group bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:border-blue-300 dark:hover:border-blue-500 transition-colors"
    >
      <div className="flex items-start justify-between">
        <h3 className="font-medium text-gray-900 dark:text-white truncate">
          {name}
        </h3>
        <span className="text-xs text-gray-500 dark:text-gray-400">
          {panes.length} pane{panes.length !== 1 ? "s" : ""}
        </span>
      </div>

      {tags.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded"
            >
              {tag}
            </span>
          ))}
        </div>
      )}

      {Object.keys(agentCounts).length > 0 && (
        <div className="mt-3 flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
          {Object.entries(agentCounts).map(([type, count]) => (
            <span key={type} className="flex items-center gap-1">
              <AgentIcon type={type} />
              <span>
                {count} {type}
              </span>
            </span>
          ))}
        </div>
      )}

      <div className="mt-3 flex items-center justify-between text-xs text-gray-400 dark:text-gray-500">
        <span>{created && `Created ${new Date(created).toLocaleDateString()}`}</span>
        <span className="text-blue-600 dark:text-blue-400 group-hover:underline">
          View →
        </span>
      </div>
    </Link>
  );
}

function AgentIcon({ type }: { type: string }) {
  // Simple colored dot for agent types
  const colors: Record<string, string> = {
    claude: "bg-orange-500",
    codex: "bg-green-500",
    gemini: "bg-blue-500",
    user: "bg-gray-500",
  };

  return (
    <span
      className={`w-2 h-2 rounded-full ${colors[type] || "bg-gray-400"}`}
    />
  );
}
