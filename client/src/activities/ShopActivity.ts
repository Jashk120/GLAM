import type { InteractionShop } from "../types/scenario";

export function showShop(
  modalEl: HTMLElement,
  overlayEl: HTMLElement,
  title: string,
  interaction: InteractionShop,
  getCoins: () => number,
  onPurchase: (itemIndex: number) => { success: boolean; message: string },
  onClose: () => void,
): void {
  function render(): void {
    const coins = getCoins();
    modalEl.innerHTML = `
      <h3>${escapeHtml(title)}</h3>
      <p>Coins: <strong>${coins} 💰</strong></p>
      <div id="shop-list"></div>
      <button class="modalBtn" id="shop-leave" style="margin-top:8px;">Leave</button>
    `;
    const list = modalEl.querySelector("#shop-list") as HTMLElement;
    interaction.items.forEach((item, idx) => {
      const row = document.createElement("div");
      row.className = "shopItem";
      const canAfford = coins >= item.price;
      row.innerHTML = `
        <div class="info">
          <span><strong>${escapeHtml(item.icon ?? "📦")} ${escapeHtml(item.name)}</strong> <span class="price">— ${item.price} 💰</span></span>
          ${item.description ? `<span style="font-size:11px;opacity:0.75">${escapeHtml(item.description)}</span>` : ""}
        </div>
      `;
      const btn = document.createElement("button");
      btn.textContent = canAfford ? "Buy" : "Too expensive";
      btn.disabled = !canAfford;
      btn.onclick = () => {
        const result = onPurchase(idx);
        // result.message shown as toast by caller; re-render on success to update coins
        void result;
        render();
      };
      row.appendChild(btn);
      list.appendChild(row);
    });
    const leave = modalEl.querySelector("#shop-leave") as HTMLButtonElement;
    leave.onclick = () => {
      overlayEl.classList.remove("active");
      onClose();
    };
  }
  render();
  overlayEl.classList.add("active");
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
