import type { IControl } from "maplibre-gl";

export class ResetViewControl implements IControl {
    private container?: HTMLDivElement;
    private button?: HTMLButtonElement;
    private icon?: HTMLElement;

    constructor(private callback: () => void, private appearance: { label: string; icon: string }) {}

    onAdd(): HTMLElement {
        this.container = document.createElement("div");
        this.container.classList.add("maplibregl-ctrl", "maplibregl-ctrl-group");
        this.button = document.createElement("button");
        this.button.type = "button";
        this.button.classList.add("reset-map-view");
        this.button.addEventListener("click", this.callback);
        this.icon = document.createElement("i");
        this.icon.setAttribute("aria-hidden", "true");
        this.button.appendChild(this.icon);
        this.container.appendChild(this.button);
        this.setAppearance(this.appearance);
        return this.container;
    }

    setAppearance(appearance: { label: string; icon: string }): void {
        this.appearance = appearance;
        if (!this.button || !this.icon) return;
        this.button.title = appearance.label;
        this.button.setAttribute("aria-label", appearance.label);
        this.icon.className = `fa ${appearance.icon} text-black`;
    }

    onRemove(): void {
        this.button?.removeEventListener("click", this.callback);
        this.container?.remove();
        this.button = undefined;
        this.container = undefined;
        this.icon = undefined;
    }
}
