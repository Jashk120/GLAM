"use client";

import { useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  FiLoader,
  FiCheckCircle,
  FiAlertCircle,
  FiCornerDownLeft,
  FiExternalLink,
  FiCopy,
  FiRefreshCw,
} from "react-icons/fi";
import { HiSparkles } from "react-icons/hi2";

const STORAGE_KEY = "glam_lastGenerated";
const GENERATE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL
    ? `${process.env.NEXT_PUBLIC_API_BASE_URL}/api/scenario/generate`
    : "http://localhost:8080/api/scenario/generate";
const SCENARIOS_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL
    ? `${process.env.NEXT_PUBLIC_API_BASE_URL}/api/scenarios`
    : "http://localhost:8080/api/scenarios";
const CLIENT_URL =
  process.env.NEXT_PUBLIC_CLIENT_URL ?? "http://localhost:5173";

type ScenarioMeta = {
  id: string;
  title: string;
  filename: string;
  generated: boolean;
};

function formatDetails(details: unknown): string {
  if (Array.isArray(details)) return (details as string[]).join("\n");
  if (typeof details === "string") return details;
  if (details) return String(details);
  return "";
}

export default function ScenarioGenerator() {
  const [prompt, setPrompt] = useState("");
  const [status, setStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [errorMsg, setErrorMsg] = useState("");
  const [errorDetails, setErrorDetails] = useState<string>("");
  const [lastGenerated, setLastGenerated] = useState<{ id: string; title: string } | null>(null);
  const [scenarios, setScenarios] = useState<ScenarioMeta[]>([]);
  const [copied, setCopied] = useState(false);

  const fetchScenarios = useCallback(async () => {
    try {
      const res = await fetch(SCENARIOS_URL);
      if (!res.ok) throw new Error(String(res.status));
      const data = (await res.json()) as { scenarios?: ScenarioMeta[] };
      setScenarios(data.scenarios ?? []);
    } catch {
      // silently keep previous list
    }
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchScenarios();
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const obj = JSON.parse(raw) as { id?: string; title?: string };
        if (obj.id) setLastGenerated({ id: obj.id, title: obj.title ?? obj.id });
      }
    } catch {
      // ignore
    }
  }, [fetchScenarios]);

  const handleGenerate = async () => {
    if (!prompt.trim() || status === "loading") return;

    setStatus("loading");
    setErrorMsg("");
    setErrorDetails("");

    try {
      const res = await fetch(GENERATE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt }),
      });

      const body = (await res.json()) as {
        scenario?: { id: string; title: string };
        error?: string;
        details?: unknown;
      };

      if (!res.ok) {
        const msg = body.error ?? `Request failed (${res.status})`;
        const details = formatDetails(body.details);
        throw new Error(details ? `${msg}\n${details}` : msg);
      }
      if (!body.scenario) {
        throw new Error("Server returned no scenario.");
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(body.scenario));
      setLastGenerated({ id: body.scenario.id, title: body.scenario.title });
      setStatus("success");
      setTimeout(() => setStatus("idle"), 5000);
      fetchScenarios();
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      const isNetwork = msg.includes("Failed to fetch") || msg.includes("NetworkError");
      if (isNetwork) {
        setErrorMsg("Network error — is the Go server running on :8080?");
      } else {
        const lines = msg.split("\n");
        setErrorMsg(lines[0] ?? msg);
        setErrorDetails(lines.slice(1).join("\n"));
      }
      setStatus("error");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      handleGenerate();
    }
  };

  const playUrl = lastGenerated ? `${CLIENT_URL}?scenario=${encodeURIComponent(lastGenerated.id)}` : "";

  const copyLink = async () => {
    if (!playUrl) return;
    try {
      await navigator.clipboard.writeText(playUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // fallback
      window.prompt("Copy link:", playUrl);
    }
  };

  return (
    <div className="w-full space-y-6">
      <div className="group relative bg-white rounded-3xl shadow-[0_8px_30px_rgb(0,0,0,0.06)] border border-zinc-200 transition-all duration-300 focus-within:ring-4 focus-within:ring-indigo-500/10 focus-within:border-indigo-400 overflow-hidden">
        <div className="p-2">
          <Textarea
            placeholder="Design a small town where students must calculate change at a bakery..."
            className="min-h-[140px] w-full resize-none border-0 shadow-none focus-visible:ring-0 text-lg leading-relaxed text-zinc-800 placeholder:text-zinc-400 bg-transparent p-4"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={status === "loading"}
            spellCheck={false}
          />
        </div>
        <div className="flex items-center justify-between bg-zinc-50/80 px-4 py-3 border-t border-zinc-100 backdrop-blur-sm">
          <div className="flex-1 flex items-center gap-2 text-sm font-medium min-w-0">
            {status === "idle" && (
              <span className="text-zinc-400 flex items-center gap-2">
                <FiCornerDownLeft className="hidden sm:block shrink-0" />
                <span className="hidden sm:inline">Press Cmd/Ctrl + Enter to generate</span>
              </span>
            )}
            {status === "loading" && (
              <span className="text-indigo-600 flex items-center gap-2 animate-pulse">
                <FiLoader className="animate-spin" size={16} /> Synthesizing world data...
              </span>
            )}
            {status === "success" && lastGenerated && (
              <span className="text-emerald-600 flex items-center gap-2">
                <FiCheckCircle size={16} className="shrink-0" /> Ready: {lastGenerated.title}
              </span>
            )}
            {status === "error" && (
              <span className="text-rose-500 flex items-center gap-2 min-w-0">
                <FiAlertCircle size={16} className="shrink-0" />
                <span className="truncate max-w-[280px] sm:max-w-[360px]" title={errorMsg}>
                  {errorMsg}
                </span>
              </span>
            )}
          </div>

          <Button
            onClick={handleGenerate}
            disabled={status === "loading" || !prompt.trim()}
            className={`rounded-xl px-6 py-5 shadow-md transition-all duration-300 shrink-0 ml-3 ${
              prompt.trim()
                ? "bg-indigo-600 hover:bg-indigo-700 text-white hover:shadow-lg hover:-translate-y-0.5"
                : "bg-zinc-200 text-zinc-400 cursor-not-allowed"
            }`}
          >
            {status === "loading" ? (
              <FiLoader className="animate-spin" size={20} />
            ) : (
              <div className="flex items-center gap-2">
                <span className="text-base font-semibold">Generate</span>
                <HiSparkles size={18} />
              </div>
            )}
          </Button>
        </div>
        {status === "error" && errorDetails && (
          <div className="px-4 pb-3">
            <pre className="text-xs bg-rose-50 border border-rose-200 rounded-xl p-3 whitespace-pre-wrap break-words text-rose-700 max-h-[160px] overflow-y-auto">
              {errorDetails}
            </pre>
          </div>
        )}
      </div>

      {lastGenerated && status === "success" && (
        <div className="bg-emerald-50 border border-emerald-200 rounded-2xl p-4 flex flex-col sm:flex-row sm:items-center gap-3 animate-in fade-in slide-in-from-bottom-2">
          <div className="flex-1 min-w-0">
            <div className="text-sm font-semibold text-emerald-800 flex items-center gap-2">
              <FiCheckCircle size={16} /> Scenario ready for students!
            </div>
            <div className="text-xs text-emerald-700 mt-1 truncate font-mono bg-white/70 px-2 py-1 rounded-lg border border-emerald-100">
              {playUrl}
            </div>
          </div>
          <div className="flex gap-2 shrink-0">
            <Button variant="outline" size="sm" onClick={copyLink} className="rounded-xl bg-white">
              <FiCopy size={14} /> {copied ? "Copied!" : "Copy link"}
            </Button>
            <Button
              size="sm"
              onClick={() => window.open(playUrl, "_blank")}
              className="rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white"
            >
              <FiExternalLink size={14} /> Play
            </Button>
          </div>
        </div>
      )}

      {lastGenerated && status !== "success" && (
        <div className="bg-white border border-zinc-200 rounded-2xl p-4 flex flex-col sm:flex-row sm:items-center gap-3">
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-zinc-700">Last generated: {lastGenerated.title}</div>
            <div className="text-xs text-zinc-500 font-mono truncate mt-1">{playUrl}</div>
          </div>
          <div className="flex gap-2 shrink-0">
            <Button variant="outline" size="sm" onClick={copyLink} className="rounded-xl">
              <FiCopy size={14} /> {copied ? "Copied!" : "Copy"}
            </Button>
            <Button size="sm" onClick={() => window.open(playUrl, "_blank")} className="rounded-xl">
              <FiExternalLink size={14} /> Open in Player
            </Button>
          </div>
        </div>
      )}

      <div className="bg-white rounded-3xl border border-zinc-200 shadow-sm overflow-hidden">
        <div className="px-5 py-4 border-b border-zinc-100 flex items-center justify-between">
          <h3 className="font-semibold text-zinc-800 flex items-center gap-2">📚 Scenarios on server</h3>
          <Button variant="ghost" size="sm" onClick={fetchScenarios} className="rounded-xl h-7 text-xs">
            <FiRefreshCw size={12} /> Refresh
          </Button>
        </div>
        {scenarios.length === 0 ? (
          <div className="px-5 py-8 text-center text-sm text-zinc-400">
            No scenarios yet. Generate one above or check the example below.
            <div className="mt-2 text-xs font-mono bg-zinc-50 inline-block px-2 py-1 rounded-lg border">💰 Money Management Town — Example</div>
          </div>
        ) : (
          <div className="divide-y divide-zinc-100">
            {scenarios.map((s) => {
              const url = `${CLIENT_URL}?scenario=${encodeURIComponent(s.id)}`;
              return (
                <div key={s.id} className="px-5 py-3 flex items-center gap-3 hover:bg-zinc-50/60 transition-colors">
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-zinc-800 truncate flex items-center gap-2">
                      {s.generated ? "✨" : "📄"} {s.title}
                      <span className="text-[11px] font-mono text-zinc-400 bg-zinc-100 px-1.5 py-0.5 rounded">{s.id}</span>
                    </div>
                    <div className="text-xs text-zinc-500 truncate">{s.filename}</div>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => window.open(url, "_blank")}
                    className="rounded-xl shrink-0"
                  >
                    <FiExternalLink size={14} /> Play
                  </Button>
                </div>
              );
            })}
          </div>
        )}
        <div className="px-5 py-3 bg-zinc-50/60 border-t border-zinc-100 text-xs text-zinc-500">
          Students play at <span className="font-mono font-medium text-zinc-700">{CLIENT_URL}</span> — use the “Play” links or share the URL with <code className="bg-white px-1 py-0.5 rounded border">?scenario=ID</code>
        </div>
      </div>
    </div>
  );
}
