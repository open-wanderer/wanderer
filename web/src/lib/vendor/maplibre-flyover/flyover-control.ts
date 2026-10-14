import type { IControl, Map } from "maplibre-gl";

export class FlyoverControl implements IControl {
    private container?: HTMLDivElement;
    private button?: HTMLButtonElement;
    private icon?: HTMLElement;

    constructor(private readonly onToggle: () => void) {}

    onAdd(_map: Map): HTMLElement {
        this.container = document.createElement("div");
        this.container.classList.add("maplibregl-ctrl", "maplibregl-ctrl-group");

        this.button = document.createElement("button");
        this.button.type = "button";
        this.button.title = "Play trail flyover";
        this.button.setAttribute("aria-label", "Play trail flyover");
        this.button.addEventListener("click", this.onToggle);

        this.icon = document.createElement("i");
        this.icon.classList.add("fa", "fa-play", "text-black");
        this.button.appendChild(this.icon);
        this.container.appendChild(this.button);

        return this.container;
    }

    onRemove(): void {
        this.button?.removeEventListener("click", this.onToggle);
        this.container?.remove();
        this.container = undefined;
        this.button = undefined;
        this.icon = undefined;
    }

    setPlaying(playing: boolean): void {
        if (!this.button || !this.icon) return;

        this.icon.classList.toggle("fa-play", !playing);
        this.icon.classList.toggle("fa-stop", playing);
        this.button.title = playing ? "Stop trail flyover" : "Play trail flyover";
        this.button.setAttribute(
            "aria-label",
            playing ? "Stop trail flyover" : "Play trail flyover",
        );
    }

    setEnabled(enabled: boolean): void {
        if (this.button) this.button.disabled = !enabled;
    }

    setVisible(visible: boolean): void {
        if (this.container) {
            this.container.style.display = visible ? "" : "none";
        }
    }
}
