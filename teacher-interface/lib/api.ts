export async function generateScenario(prompt: string) {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
  const response = await fetch(`${base}/api/scenario/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prompt }),
  });

  if (!response.ok) {
    throw new Error(`Generation failed: ${response.statusText}`);
  }

  return response.json();
}
