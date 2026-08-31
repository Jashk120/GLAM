import type { InteractionMath } from "../types/scenario";

export function showMath(
  modalEl: HTMLElement,
  overlayEl: HTMLElement,
  title: string,
  interaction: InteractionMath,
  onAnswer: (correct: boolean) => void,
): void {
  const hintHtml = interaction.hint ? `<div class="hint">💡 Hint: ${escapeHtml(interaction.hint)}</div>` : "";
  modalEl.innerHTML = `
    <h3>${escapeHtml(title)}</h3>
    <p>${escapeHtml(interaction.question)}</p>
    ${hintHtml}
    <input id="mathInput" type="text" placeholder="Your answer" autocomplete="off" />
    <button class="modalBtn" id="math-submit">Submit</button>
    <button class="modalBtn" id="math-cancel">Cancel</button>
    <div id="math-feedback" class="math-feedback"></div>
  `;
  overlayEl.classList.add("active");
  const input = modalEl.querySelector("#mathInput") as HTMLInputElement;
  const feedback = modalEl.querySelector("#math-feedback") as HTMLElement;
  const submit = modalEl.querySelector("#math-submit") as HTMLButtonElement;
  const cancel = modalEl.querySelector("#math-cancel") as HTMLButtonElement;

  input.focus();

  function check(): void {
    const raw = input.value.trim();
    if (raw.length === 0) {
      feedback.textContent = "Please enter an answer.";
      return;
    }
    const correct = isAnswerCorrect(raw, interaction.answer, interaction.tolerance ?? 0);
    overlayEl.classList.remove("active");
    onAnswer(correct);
  }

  submit.onclick = check;
  cancel.onclick = () => overlayEl.classList.remove("active");
  input.addEventListener("keydown", (e) => {
    e.stopPropagation();
    if (e.key === "Enter") { e.preventDefault(); check(); }
    else if (e.key === "Escape") { e.preventDefault(); overlayEl.classList.remove("active"); }
    else if (["ArrowUp","ArrowDown","ArrowLeft","ArrowRight","w","a","s","d","W","A","S","D","e","E"," "].includes(e.key)) {
      e.stopImmediatePropagation?.();
    }
  });
}

function isAnswerCorrect(input: string, answer: number | string, tolerance: number): boolean {
  if (typeof answer === "number") {
    const num = Number(input);
    if (Number.isNaN(num)) return false;
    if (tolerance > 0) return Math.abs(num - answer) <= tolerance;
    return num === answer;
  }
  return input.trim().toLowerCase() === String(answer).trim().toLowerCase();
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
