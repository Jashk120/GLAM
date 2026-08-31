import ScenarioGenerator from "@/components/teacher/ScenarioGenerator";

export default function TeacherDashboard() {
  return (
    <main className="min-h-screen bg-zinc-50">
      <header className="sticky top-0 z-20 backdrop-blur-xl bg-white/80 border-b border-zinc-200">
        <div className="max-w-3xl mx-auto px-4 md:px-8 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-xl">🧑‍🏫</span>
            <span className="font-bold text-zinc-900 tracking-tight">GLAM Teacher</span>
            <span className="hidden sm:inline text-xs font-medium text-zinc-500 bg-zinc-100 px-2 py-1 rounded-full border">Phase 7 → 8</span>
          </div>
          <a
            href="http://localhost:5173"
            target="_blank"
            rel="noreferrer"
            className="text-xs font-medium text-indigo-600 hover:text-indigo-700 flex items-center gap-1.5 bg-indigo-50 hover:bg-indigo-100 px-3 py-1.5 rounded-full border border-indigo-200 transition-colors"
          >
            🎮 Open Player <span aria-hidden>↗</span>
          </a>
        </div>
      </header>

      <div className="max-w-3xl mx-auto px-4 md:px-8 py-8 md:py-10">
        <div className="mb-8">
          <h1 className="text-2xl md:text-3xl font-bold tracking-tight text-zinc-900">Create a learning world</h1>
          <p className="text-sm md:text-[15px] leading-relaxed text-zinc-500 mt-2 max-w-2xl">
            Describe your lesson in plain language. GLAM generates a playable Phaser world validated against the schema — then students play it instantly at <span className="font-mono text-zinc-700 bg-white px-1.5 py-0.5 rounded border text-xs">localhost:5173</span>.
          </p>
        </div>

        <ScenarioGenerator />

        <p className="text-center text-xs text-zinc-400 mt-8">
          Powered by Go validator + OpenRouter LLM · No teacher UI in the player — pure student experience.
        </p>
      </div>
    </main>
  );
}
