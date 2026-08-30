"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { 
  FiLoader, 
  FiCheckCircle, 
  FiAlertCircle, 
  FiCornerDownLeft 
} from "react-icons/fi";
import {   HiSparkles  } from "react-icons/hi2"

const STORAGE_KEY = "glam_lastGenerated";
const GENERATE_URL = "http://localhost:8080/api/scenario/generate";

export default function ScenarioGenerator() {
  const [prompt, setPrompt] = useState("");
  const [status, setStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [errorMsg, setErrorMsg] = useState("");

  const handleGenerate = async () => {
    if (!prompt.trim() || status === "loading") return;

    setStatus("loading");
    setErrorMsg("");

    try {
      const res = await fetch(GENERATE_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt }),
      });

      const body = await res.json();

      if (!res.ok) {
        throw new Error(body.error ?? `Request failed (${res.status})`);
      }
      if (!body.scenario) {
        throw new Error("Server returned no scenario.");
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(body.scenario));
      setStatus("success");

      setTimeout(() => setStatus("idle"), 4000);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      setErrorMsg(msg.includes("Failed to fetch") ? "Network error: Is the server running?" : msg);
      setStatus("error");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      handleGenerate();
    }
  };

  return (
    <div className="group relative bg-white rounded-3xl shadow-[0_8px_30px_rgb(0,0,0,0.04)] border border-zinc-200 transition-all duration-300 focus-within:ring-4 focus-within:ring-indigo-500/10 focus-within:border-indigo-400 overflow-hidden">
      
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
        <div className="flex-1 flex items-center gap-2 text-sm font-medium">
          {status === "idle" && (
            <span className="text-zinc-400 flex items-center gap-2">
              <FiCornerDownLeft className="hidden sm:block" />
              <span className="hidden sm:inline">Press Cmd/Ctrl + Enter to generate</span>
            </span>
          )}
          {status === "loading" && (
            <span className="text-indigo-600 flex items-center gap-2 animate-pulse">
              <FiLoader className="animate-spin" size={16} /> Synthesizing world data...
            </span>
          )}
          {status === "success" && (
            <span className="text-emerald-600 flex items-center gap-2 animate-in fade-in slide-in-from-bottom-2">
              <FiCheckCircle size={16} /> Scenario beamed to game client
            </span>
          )}
          {status === "error" && (
            <span className="text-rose-500 flex items-center gap-2 animate-in fade-in">
              <FiAlertCircle size={16} className="shrink-0" />
              <span className="truncate max-w-[300px]" title={errorMsg}>{errorMsg}</span>
            </span>
          )}
        </div>

        <Button 
          onClick={handleGenerate} 
          disabled={status === "loading" || !prompt.trim()}
          className={`rounded-xl px-6 py-5 shadow-md transition-all duration-300 ${
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
    </div>
  );
}
