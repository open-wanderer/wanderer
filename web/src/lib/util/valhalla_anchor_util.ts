interface ValhallaAnchorDisplay {
    icon: string;
    number: number | null;
    titleKey: "start" | "finish" | "route-point";
}

export function valhallaAnchorDisplay(index: number, total: number): ValhallaAnchorDisplay {
    if (index === 0) {
        return {
            icon: "fa-bullseye",
            number: null,
            titleKey: "start",
        };
    }

    if (index === total - 1) {
        return {
            icon: "fa-flag-checkered",
            number: null,
            titleKey: "finish",
        };
    }

    return {
        icon: "fa-location-dot",
        number: index,
        titleKey: "route-point",
    };
}

export function valhallaAnchorTitle(
    index: number,
    total: number,
    translate: (key: string) => string,
) {
    const display = valhallaAnchorDisplay(index, total);
    if (display.number === null) {
        return translate(display.titleKey);
    }

    return `${translate(display.titleKey)} #${display.number}`;
}

export function renderValhallaAnchorMarker(
    element: HTMLElement,
    index: number,
    total: number,
) {
    const display = valhallaAnchorDisplay(index, total);
    element.replaceChildren();

    if (display.number !== null) {
        element.textContent = `${display.number}`;
        return;
    }

    const icon = document.createElement("i");
    icon.classList.add("fa", display.icon);
    element.appendChild(icon);
}
