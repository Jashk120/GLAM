import type { InteractionMCQ } from "../types/scenario";

export function showMCQ(
  modalEl: HTMLElement,
  overlayEl: HTMLElement,
  title: string,
  interaction: InteractionMCQ,
  onAnswer: (correct: boolean) => void,
): void {
  modalEl.innerHTML = `
    <h3>${escapeHtml(title)}</h3>
    <p>${escapeHtml(interaction.question)}</p>
    <div id="mcq-options"></div>
  `;
  const container = modalEl.querySelector("#mcq-options") as HTMLElement;
  interaction.options.forEach((opt, idx) => {
    const btn = document.createElement("button");
    btn.className = "modalBtn";
    btn.textContent = opt.text;
    btn.onclick = () => {
      overlayEl.classList.remove("active");
      onAnswer(opt.correct);
    };
    void idx;
    container.appendChild(btn);
  });
  overlayEl.classList.add("active");
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
