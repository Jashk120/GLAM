import type { InteractionInformation } from "../types/scenario";

export function showInformation(
  modalEl: HTMLElement,
  overlayEl: HTMLElement,
  interaction: InteractionInformation,
  onClose: () => void,
): void {
  const title = interaction.title ?? "Information";
  modalEl.innerHTML = `
    <h3>${escapeHtml(title)}</h3>
    <p style="white-space:pre-wrap">${escapeHtml(interaction.content)}</p>
    ${interaction.image ? `<div style="font-size:11px;opacity:0.6;margin-bottom:10px">🖼️ ${escapeHtml(interaction.image)}</div>` : ""}
    <button class="modalBtn" id="info-ok">OK</button>
  `;
  overlayEl.classList.add("active");
  const btn = modalEl.querySelector("#info-ok") as HTMLButtonElement;
  btn.focus();
  btn.onclick = () => {
    overlayEl.classList.remove("active");
    onClose();
  };
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;").replace(/`/g, "&#96;");
}
