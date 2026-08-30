"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { FaWandMagicSparkles, FaSpinner, FaCircleCheck, FaTriangleExclamation } from "react-icons/fa6";

// Formatting helper exactly as it was in your TS file
function formatDetails(details: unknown): string {
  if (Array.isArray(details)) return details.map((d) => `• ${String(d)}`).join("\n");
  if (typeof details === "string") return details;
  if (details) return String(details);
  return "";
}

const STORAGE_KEY = "glam_lastGenerated";
const GENERATE_URL = "http://localhost:8080/api/scenario/generate"; // Make sure to use the absolute URL for your Go server

export default function ScenarioGenerator() {
  const [prompt, setPrompt] = useState("");
  const [status, setStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [errorMsg, setErrorMsg] = useState("");
  const [generatedTitle, setGeneratedTitle] = useState("");

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
        const msg = body.error ?? `Request failed (${res.status})`;
        const details = body.details;
        throw new Error(details ? `${msg}\n${formatDetails(details)}` : msg);
      }

      if (!body.scenario) {
        throw new Error("Server returned no scenario.");
      }

      const scenario = body.scenario;

      // Note: In Next.js, we assume the Go backend JSON schema validation is sufficient.
      // The Phaser client will run its own `validateScenarioSync` when it boots it.
      
      // Save to localStorage (The Bridge to your Phaser game)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(scenario));
      
      setGeneratedTitle(scenario.title || "New Scenario");
      setStatus("success");

      // Reset success message after 3 seconds (replicating window.setTimeout from your TS)
      setTimeout(() => {
        setStatus("idle");
      }, 3000);

    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      
      if (msg.includes("Failed to fetch") || msg.includes("NetworkError")) {
        setErrorMsg("Network error — is the Go server running on :8080?");
      } else {
        setErrorMsg(`Error: ${msg}`);
      }
      
      setStatus("error");
    }
  };

  // Replicating the Ctrl+Enter / Cmd+Enter shortcut
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      handleGenerate();
    }
  };

  return (
    <Card className="w-full max-w-2xl border-slate-200 shadow-sm">
      <CardHeader>
        <CardTitle className="text-2xl flex items-center gap-2">
          Generate Lesson
        </CardTitle>
        <CardDescription>
          Describe your pedagogical goal. AI will generate a playable world.
        </CardDescription>
      </CardHeader>
      
      <CardContent className="space-y-4">
        <Textarea
          placeholder="e.g., Create a small town where students learn basic money management..."
          className="min-h-[120px] resize-none focus-visible:ring-blue-500"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={status === "loading"}
        />
        
        <div className="flex items-center justify-between min-h-[40px]">
          <Button 
            onClick={handleGenerate} 
            disabled={status === "loading" || !prompt.trim()}
            className="w-48 transition-all bg-blue-600 hover:bg-blue-700"
          >
            {status === "loading" ? (
              <><FaSpinner className="mr-2 animate-spin" /> Generating...</>
            ) : (
              <><FaWandMagicSparkles className="mr-2" /> Generate Scenario</>
            )}
          </Button>

          {/* Status Indicators replacing genStatus & genError logic */}
          <div className="text-sm font-medium flex-1 text-right ml-4">
            {status === "success" && (
              <span className="flex items-center justify-end text-green-600 gap-2 animate-in fade-in">
                <FaCircleCheck /> Generated! Loaded "{generatedTitle}" into game.
              </span>
            )}
            
            {status === "error" && (
              <span className="flex items-center justify-end text-red-600 gap-2 whitespace-pre-line text-left">
                <FaTriangleExclamation className="shrink-0" /> 
                <span>{errorMsg}</span>
              </span>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
