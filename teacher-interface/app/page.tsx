import ScenarioGenerator from "@/components/teacher/ScenarioGenerator";

export default function TeacherDashboard() {
  return (
    <main className="min-h-screen bg-slate-50 p-8 md:p-16">
      <div className="max-w-5xl mx-auto space-y-8">
        
        <header>
          <h1 className="text-4xl font-bold tracking-tight text-slate-900">
            GLAM Teacher Interface
          </h1>
          <p className="text-slate-500 mt-2 text-lg">
            Design scenarios, validate layouts, and stream assets to your students.
          </p>
        </header>

        <section>
          <ScenarioGenerator />
        </section>

      </div>
    </main>
  );
}

