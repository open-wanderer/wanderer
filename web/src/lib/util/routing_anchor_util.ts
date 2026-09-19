interface RoutingAnchorDisplay {
    icon: string;
    number: number | null;
    titleKey: "start" | "finish" | "route-point";
    isStartAndFinish: boolean;
}

export interface RoutingAnchorListEntry<T> {
    anchor: T;
    anchorIndex: number;
    isFinishCopy: boolean;
}

export function routingAnchorListEntries<T>(
    anchors: readonly T[],
    closedLoop = false,
): RoutingAnchorListEntry<T>[] {
    const entries = anchors.map((anchor, anchorIndex) => ({
        anchor,
        anchorIndex,
        isFinishCopy: false,
    }));

    if (closedLoop && anchors.length > 0) {
        entries.push({
            anchor: anchors[0],
            anchorIndex: 0,
            isFinishCopy: true,
        });
    }

    return entries;
}

export function routingAnchorDisplay(
    index: number,
    total: number,
    closedLoop = false,
): RoutingAnchorDisplay {
    if (index === 0) {
        return {
            icon: "fa-bullseye",
            number: null,
            titleKey: "start",
            isStartAndFinish: closedLoop,
        };
    }

    if (!closedLoop && index === total - 1) {
        return {
            icon: "fa-flag-checkered",
            number: null,
            titleKey: "finish",
            isStartAndFinish: false,
        };
    }

    return {
        icon: "fa-location-dot",
        number: index,
        titleKey: "route-point",
        isStartAndFinish: false,
    };
}

export function routingAnchorTitle(
    index: number,
    total: number,
    translate: (key: string) => string,
    closedLoop = false,
) {
    const display = routingAnchorDisplay(index, total, closedLoop);
    if (display.isStartAndFinish) {
        return `${translate("start")} / ${translate("finish")}`;
    }
    if (display.number === null) {
        return translate(display.titleKey);
    }

    return `${translate(display.titleKey)} #${display.number}`;
}

export function renderRoutingAnchorMarker(
    element: HTMLElement,
    index: number,
    total: number,
    closedLoop = false,
) {
    const display = routingAnchorDisplay(index, total, closedLoop);
    element.replaceChildren();

    if (display.number !== null) {
        element.textContent = `${display.number}`;
        return;
    }

    const icon = document.createElement("i");
    icon.classList.add("fa", display.icon);
    element.appendChild(icon);
}
