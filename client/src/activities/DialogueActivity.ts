import type { InteractionDialogue } from "../types/scenario";

export type DialogueResult = { closed: boolean };

export function showDialogue(
  modalEl: HTMLElement,
  overlayEl: HTMLElement,
  speaker: string,
  interaction: InteractionDialogue,
  onClose: () => void,
): void {
  const title = interaction.speaker ?? speaker;
  modalEl.innerHTML = `
    <h3>${escapeHtml(title)}</h3>
    <p>${escapeHtml(interaction.text)}</p>
    <button class="modalBtn" id="dlg-ok">OK</button>
  `;
  overlayEl.classList.add("active");
  const btn = modalEl.querySelector("#dlg-ok") as HTMLButtonElement;
  btn.focus();
  btn.onclick = () => {
    overlayEl.classList.remove("active");
    onClose();
  };
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
