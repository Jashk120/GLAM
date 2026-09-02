import { ArenaFlow } from "./ArenaFlow";
import { consoleModel, stageObjectCount } from "./ArenaPresentation";
import type { ArenaAnswerResult, ArenaNode, ArenaScript, TeachingNode } from "./arenaTypes";

const OBJECT_ICONS: Record<TeachingNode["stage"]["visual"]["object"], string> = {
  apple: "🍎", coin: "🪙", star: "⭐",
};

const EXPRESSION_ICON: Record<string, string> = {
  neutral: "🦊", happy: "🦊", thinking: "🤔", encouraging: "🦊", celebrating: "🎉", concerned: "🦊",
};

function element<K extends keyof HTMLElementTagNameMap>(tag: K, className?: string): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag);
  if (className) el.className = className;
  return el;
}

export class ArenaRuntime {
  private readonly flow: ArenaFlow;
  private selectedOptionId: string | null = null;
  private revealedStage = false;
  private lastTeachingNode: TeachingNode | null = null;
  private feedback: ArenaAnswerResult | null = null;

  constructor(private readonly root: HTMLElement, private readonly script: ArenaScript) {
    this.flow = new ArenaFlow(script);
  }

  mount(): void {
    this.root.hidden = false;
    this.render();
  }

  destroy(): void {
    this.root.replaceChildren();
    this.root.hidden = true;
  }

  private currentTeachingNode(): TeachingNode | null {
    const node = this.flow.current;
    if (node.type === "teaching") this.lastTeachingNode = node;
    return node.type === "teaching" ? node : this.lastTeachingNode;
  }

  private render(): void {
    const node = this.flow.current;
    const shell = element("section", `arenaShell arenaTheme-${this.script.theme}`);
    shell.setAttribute("aria-label", `${this.script.title} learning arena`);
    shell.append(this.renderHeader(), this.renderStage(), this.renderConsole(node));
    this.root.replaceChildren(shell);
  }

  private renderHeader(): HTMLElement {
    const header = element("header", "arenaHeader");
    const title = element("strong");
    title.textContent = this.script.title;
    const step = element("span", "arenaStep");
    step.textContent = `Learning arena · ${this.flow.current.id}`;
    header.append(title, step);
    return header;
  }

  private renderStage(): HTMLElement {
    const stage = element("section", "arenaStage");
    stage.setAttribute("aria-label", "Teaching stage");
    const teaching = this.currentTeachingNode();
    const visual = element("div", "arenaVisual");
    if (teaching) {
      const count = stageObjectCount(teaching, this.revealedStage);
      visual.setAttribute("aria-label", `${count} ${teaching.stage.visual.object}${count === 1 ? "" : "s"}`);
      for (let i = 0; i < count; i++) {
        const icon = element("span", "arenaObject");
        icon.textContent = OBJECT_ICONS[teaching.stage.visual.object];
        if (this.revealedStage && i >= teaching.stage.visual.count) icon.classList.add("arenaObjectAdded");
        visual.appendChild(icon);
      }
      if (!this.revealedStage && (teaching.stage.motion?.length ?? 0) > 0) {
        const show = element("button", "arenaReplayButton");
        show.type = "button";
        show.textContent = "Show the change";
        show.addEventListener("click", () => {
          this.revealedStage = true;
          this.render();
        });
        visual.appendChild(show);
      }
    } else {
      visual.textContent = "✨";
      visual.setAttribute("aria-label", "Learning arena ready");
    }
    stage.append(this.renderAvatar("student"), visual, this.renderAvatar("mascot"));
    return stage;
  }

  private renderAvatar(role: "student" | "mascot"): HTMLElement {
    const figure = element("div", `arenaAvatar arenaAvatar-${role}`);
    const icon = element("span", "arenaAvatarIcon");
    const label = element("span", "arenaAvatarLabel");
    if (role === "student") {
      const config = this.script.cast.student;
      icon.textContent = config.variant === "girl" ? "👧" : config.variant === "boy" ? "🧒" : "🧑";
      label.textContent = config.name ?? "Student";
    } else {
      const config = this.script.cast.mascot;
      icon.textContent = EXPRESSION_ICON[config.expression ?? "neutral"];
      label.textContent = config.name;
    }
    figure.append(icon, label);
    return figure;
  }

  private renderConsole(node: ArenaNode): HTMLElement {
    const consoleEl = element("section", "arenaConsole");
    consoleEl.setAttribute("aria-live", "polite");
    const title = element("div", "arenaConsoleTitle");
    title.textContent = this.feedback ? "Nova says" : "Learning console";
    const body = element("div", "arenaConsoleBody");
    if (this.feedback) {
      const message = element("p", this.feedback.correct ? "arenaFeedback arenaFeedbackCorrect" : "arenaFeedback arenaFeedbackWrong");
      message.textContent = this.feedback.feedback.mascotText;
      const next = element("button", "arenaPrimaryButton");
      next.type = "button";
      next.textContent = this.feedback.correct ? "Continue" : "Try again";
      next.addEventListener("click", () => {
        this.feedback = null;
        this.selectedOptionId = null;
        this.render();
      });
      body.append(message, next);
    } else {
      const model = consoleModel(node);
      if (model.kind === "reading") {
        const text = element("p", "arenaReadText");
        text.textContent = model.text;
        const controls = element("div", "arenaControls");
        if (node.type === "teaching" && node.stage.replayable) {
          const replay = element("button", "arenaSecondaryButton");
          replay.type = "button";
          replay.textContent = "Replay visual";
          replay.addEventListener("click", () => {
            this.revealedStage = false;
            this.render();
          });
          controls.appendChild(replay);
        }
        const next = element("button", "arenaPrimaryButton");
        next.type = "button";
        next.textContent = model.continueLabel;
        next.addEventListener("click", () => {
          this.revealedStage = node.type === "teaching" ? true : this.revealedStage;
          this.flow.continue();
          this.render();
        });
        controls.appendChild(next);
        body.append(text, controls);
      } else if (model.kind === "multipleChoice") {
        const question = element("p", "arenaQuestion");
        question.textContent = model.prompt;
        const choices = element("div", "arenaChoices");
        for (const option of model.options) {
          const choice = element("button", "arenaChoice");
          choice.type = "button";
          choice.textContent = option.text;
          choice.setAttribute("aria-pressed", String(this.selectedOptionId === option.id));
          if (this.selectedOptionId === option.id) choice.classList.add("arenaChoiceSelected");
          choice.addEventListener("click", () => {
            this.selectedOptionId = option.id;
            this.render();
          });
          choices.appendChild(choice);
        }
        const submit = element("button", "arenaPrimaryButton");
        submit.type = "button";
        submit.textContent = "Check answer";
        submit.disabled = this.selectedOptionId === null;
        submit.addEventListener("click", () => {
          if (!this.selectedOptionId) return;
          this.feedback = this.flow.submitAnswer([this.selectedOptionId]);
          this.render();
        });
        body.append(question, choices, submit);
      } else {
        const heading = element("h2", "arenaCompleteTitle");
        heading.textContent = model.title;
        const summary = element("p", "arenaReadText");
        summary.textContent = model.summary;
        body.append(heading, summary);
      }
    }
    consoleEl.append(title, body);
    return consoleEl;
  }
}
