import ScenarioGenerator from "@/components/teacher/ScenarioGenerator";
export default function TeacherDashboard() {
  return (
    <main className="min-h-screen relative flex flex-col items-center justify-center bg-zinc-50 overflow-hidden px-4 md:px-8">
      <div className="relative z-10 w-full max-w-3xl flex flex-col items-center space-y-12 mb-20">
        <div className="w-full">
          <ScenarioGenerator />
        </div>

      </div>
    </main>
  );
}

