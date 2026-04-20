<script lang="ts">
    import { page } from "$app/state";
    import "$lib/assets/fonts/IBMPlexSans-Regular-normal";
    import "$lib/assets/fonts/IBMPlexSans-SemiBold-bold";
    import "$lib/assets/fonts/fa-solid-900-normal";
    import Button from "$lib/components/base/button.svelte";
    import Select from "$lib/components/base/select.svelte";
    import LogoText from "$lib/components/logo/logo_text.svelte";
    import MapWithElevationMaplibre from "$lib/components/trail/map_with_elevation_maplibre.svelte";
    import type { Settings } from "$lib/models/settings";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { trail } from "$lib/stores/trail_store";
    import {
        formatDistance,
        formatElevation,
        formatHTMLAsText,
        formatTimeHHMM,
    } from "$lib/util/format_util";
    import {
        calculatePixelPerMeter,
        calculateScaleFactor,
    } from "$lib/util/maplibre_util";

    import { createRect, createText } from "$lib/util/svg_util";
    import QrCodeWithLogo from "$lib/vendor/qr-code-with-logos/index";
    import { jsPDF } from "jspdf";
    import * as M from "maplibre-gl";
    import { onMount, tick } from "svelte";
    import { _ } from "svelte-i18n";

    let map: M.Map | undefined = $state();
    let showGrid: boolean = $state(true);

    const settings: Settings = page.data.settings;

    const paperSizes: { text: string; value: keyof typeof paperDimensions }[] =
        [
            { text: "A4", value: "a4" },
            { text: "Letter", value: "letter" },
        ];
    const paperDimensions = {
        a4: {
            css: {
                width: "21cm",
                height: "29.7cm",
            },
            pdf: {
                width: 210,
                height: 297,
            },
        },
        letter: {
            css: {
                width: "21.59cm",
                height: "27.94cm",
            },
            pdf: {
                width: 215.9,
                height: 279.4,
            },
        },
    };
    let selectedPaperSize = $state(paperSizes[0].value);

    const orientations: { text: string; value: "portrait" | "landscape" }[] = [
        { text: "Portrait", value: "portrait" },
        { text: "Landscape", value: "landscape" },
    ];
    let selectedOrientation: "portrait" | "landscape" = $state(
        orientations[0].value,
    );

    const gridOptions = [
        { text: $_("degrees"), value: "degree" },
        { text: $_("no-grid"), value: "off" },
    ];
    let selectedGrid = $state(gridOptions[0].value);

    let scale: number = $state(1);

    let printLoading: boolean = $state(false);

    let includeDescription: boolean = $state(false);
    let includeWaypoints: boolean = $state(false);

    onMount(() => {
        let qrcode = new QrCodeWithLogo({
            content: page.url.href.replace("/print", ""),
            image: document.getElementById("qrcode") as HTMLImageElement,
            logo: {
                src: "/favicon.png",
            },
        });
    });

    function faUnicode(name: string) {
        var testI = document.createElement("i");
        var char;

        testI.className = "fa fa-" + name;
        document.body.appendChild(testI);

        char = window
            .getComputedStyle(testI, ":before")
            .content.replace(/'|"/g, "");

        testI.remove();

        return char.charCodeAt(0);
    }

    function getTextHeight(text: string, doc: jsPDF, width: number) {
        // Split text by newlines first to preserve blank lines
        const paragraphs = text.split('\n');
        let numLines = 0;

        for (const paragraph of paragraphs) {
            if (paragraph === '') {
                // Empty line (blank line between paragraphs)
                numLines += 1;
            } else {
                // Non-empty paragraph - count wrapped lines
                numLines += doc.splitTextToSize(paragraph, width).length;
            }
        }

        return (
            (numLines * doc.getFontSize() * doc.getLineHeightFactor() -
                doc.getFontSize() * (doc.getLineHeightFactor() - 1)) /
            doc.internal.scaleFactor
        );
    }

    async function print() {
        if (!map) {
            return;
        }
        printLoading = true;
        const doc = new jsPDF(selectedOrientation, "mm", [
            paperDimensions[selectedPaperSize].pdf[
                selectedOrientation == "portrait" ? "width" : "height"
            ],
            paperDimensions[selectedPaperSize].pdf[
                selectedOrientation == "portrait" ? "height" : "width"
            ],
        ]);
        doc.setFont("IBMPlexSans-Regular", "normal");
        var width = doc.internal.pageSize.getWidth();
        var height = doc.internal.pageSize.getHeight();
        let currentHeight = 0;
        const rulerElement = document.getElementById("ruler") as HTMLElement;
        const rulerSegments = rulerElement.getElementsByTagName("rect");
        const rulerLabels = rulerElement.getElementsByTagName("text");

        const pageWidth = document
            .getElementById("paper")
            ?.getBoundingClientRect().width;

        try {
            var img = document.createElement("img");
            let dimensions = map.getCanvas().getBoundingClientRect();
            img.width = dimensions.width;
            img.height = dimensions.height;
            let ratio = img.height / img.width;
            img.src = map.getCanvas().toDataURL();

            const imgPadding = 8;
            const docImgWidth = width - 2 * imgPadding;
            const docImgHeight = docImgWidth * ratio;
            doc.addImage(
                img.src,
                "png",
                imgPadding,
                imgPadding,
                0,
                docImgHeight,
            );
            currentHeight += docImgHeight + 11;
            const elevationProfileHeight = currentHeight + 6 - 11;

            // Waypoints
            const canvasContainer = map.getCanvasContainer();
            const marker =
                canvasContainer.getElementsByClassName("maplibregl-marker");
            for (const m of marker) {
                if (m.id === "elevation-marker") {
                    continue;
                }
                const markerStyle = window.getComputedStyle(m);
                const matrix = new DOMMatrixReadOnly(markerStyle.transform);
                const percentX = matrix.m41 / dimensions.width;
                const percentY = matrix.m42 / dimensions.height;

                const markerIcon = m.querySelector("i") as HTMLElement;
                const iconContent = window
                    .getComputedStyle(markerIcon, ":before")
                    .getPropertyValue("content");
                const iconText = iconContent.replaceAll('"', "");

                const iconPositionX = docImgWidth * percentX + imgPadding;
                const iconPositionY = docImgHeight * percentY + imgPadding;
                const circleRadius = 3;

                doc.setFillColor(107, 114, 128);
                doc.circle(
                    iconPositionX + circleRadius,
                    iconPositionY + circleRadius,
                    circleRadius,
                    "F",
                );
                doc.setTextColor("#ffffff");
                doc.setFontSize(7);
                doc.setFont("fa-solid-900", "normal");

                const { w: letterWidth, h: letterHeight } =
                    doc.getTextDimensions(iconText);

                doc.text(
                    iconText,
                    iconPositionX + 2 * circleRadius - letterWidth / 2 - 0.3,
                    iconPositionY + circleRadius + letterHeight / 2 - 0.33,
                );
            }

            // QR Code
            doc.addImage(
                document.getElementById("qrcode") as HTMLImageElement,
                "png",
                8,
                currentHeight,
                24,
                24,
            );

            // Distance
            currentHeight += 5;
            let currentWidth = 8 + 24 + 4;
            doc.setFontSize(9);
            doc.setTextColor(107, 114, 128);
            doc.setFont("fa-solid-900", "normal");
            doc.text("\uf337", currentWidth, currentHeight);
            currentWidth += doc.getTextWidth("\uf337") + 2;
            doc.setFont("IBMPlexSans-Regular", "normal");
            doc.text(
                formatDistance($trail.distance),
                currentWidth,
                currentHeight,
            );
            currentWidth +=
                doc.getTextWidth(formatDistance($trail.distance)) + 4;

            // Duration
            doc.setFont("fa-solid-900", "normal");
            doc.text("\uf017", currentWidth, currentHeight);
            currentWidth += doc.getTextWidth("\uf337") + 2;
            doc.setFont("IBMPlexSans-Regular", "normal");
            doc.text(
                formatTimeHHMM($trail.duration),
                currentWidth,
                currentHeight,
            );
            currentWidth +=
                doc.getTextWidth(formatTimeHHMM($trail.duration)) + 4;

            // Elevation gain
            doc.setFont("fa-solid-900", "normal");
            doc.text("\ue098", currentWidth, currentHeight);
            currentWidth += doc.getTextWidth("\ue098") + 2;
            doc.setFont("IBMPlexSans-Regular", "normal");
            doc.text(
                formatElevation($trail.elevation_gain),
                currentWidth,
                currentHeight,
            );
            currentWidth +=
                doc.getTextWidth(formatDistance($trail.elevation_gain)) + 4;

            // Elevation loss
            doc.setFont("fa-solid-900", "normal");
            doc.text("\ue097", currentWidth, currentHeight);
            currentWidth += doc.getTextWidth("\ue098") + 2;
            doc.setFont("IBMPlexSans-Regular", "normal");
            doc.text(
                formatElevation($trail.elevation_loss),
                currentWidth,
                currentHeight,
            );

            // Ruler
            currentHeight += 6;
            currentWidth = 8 + 24 + 4;
            doc.setTextColor(0, 0, 0);
            for (let i = 0; i < rulerSegments.length; i++) {
                const segment = rulerSegments[i];
                const label = rulerLabels[i].textContent ?? "";
                const segmentWidth = segment.getBoundingClientRect().width;
                const segmentWidthPercent = segmentWidth / (pageWidth ?? 0);
                const segmentPDFWidth = width * segmentWidthPercent;
                let textWidth = doc.getTextWidth(label);

                if (i % 2 == 0) {
                    doc.setFillColor(255, 255, 255);
                } else {
                    doc.setFillColor(93, 93, 93);
                }

                doc.rect(currentWidth, currentHeight, segmentPDFWidth, 2, "FD");
                doc.text(
                    label,
                    currentWidth - textWidth / 2,
                    currentHeight + 6,
                );
                currentWidth += segmentPDFWidth;
                if (i == rulerSegments.length - 1) {
                    const endLabel = rulerLabels[i + 1].textContent ?? "";
                    textWidth = doc.getTextWidth(endLabel);
                    doc.text(
                        endLabel,
                        currentWidth - textWidth / 2,
                        currentHeight + 6,
                    );
                    textWidth = doc.getTextWidth("KM");
                    doc.text("KM", currentWidth - textWidth, currentHeight - 2);
                }
            }

            // Scale
            currentHeight += 12;
            doc.setFillColor(255, 255, 255);
            doc.text(
                `Scale:  1 : ${Math.round(scale)}`,
                8 + 24 + 4,
                currentHeight,
            );

            // Elevation profile
            const elevationChartCanvas = document.getElementById(
                "elevation-profile-chart",
            ) as HTMLCanvasElement;
            dimensions = elevationChartCanvas.getBoundingClientRect();
            img.width = dimensions.width;
            img.height = dimensions.height;
            ratio = img.height / img.width;
            img.src = elevationChartCanvas.toDataURL();

            doc.addImage(
                elevationChartCanvas.toDataURL("image/png"),
                width - 79.3 - 8,
                elevationProfileHeight,
                79.3,
                79.3 * ratio,
            );

            // Separator
            currentHeight += 5;
            doc.setDrawColor(232, 234, 237);
            doc.line(8, currentHeight, width - 8, currentHeight);
            currentHeight += 6;

            // Trail name & location
            doc.setTextColor(0, 0, 0);
            doc.setFont("IBMPlexSans-SemiBold", "bold");
            doc.text($trail.name, 8, currentHeight);
            doc.setFont("fa-solid-900", "normal");
            currentHeight += 5;
            doc.text("\uf3c5", 8, currentHeight);
            doc.setFont("IBMPlexSans-Regular", "normal");
            doc.text($trail.location || "-", 12, currentHeight);

            // End of page
            currentHeight = height;

            // Logo
            const logo = new Image();
            logo.src = "/imgs/logo_text_dark.png";
            const logoRatio = 64 / 212;
            const logoWidth = 32;
            const logoHeight = logoWidth * logoRatio;

            function addLogo() {
                doc.addImage(
                    logo,
                    width - 8 - logoWidth,
                    height - 4 - logoHeight,
                    logoWidth,
                    logoHeight,
                );
            }

            function newPage() {
                addLogo();
                doc.addPage();
                currentHeight = 16;
            }

            if (includeDescription && $trail.description) {
                const descriptionText = formatHTMLAsText($trail.description);
                let textHeight = getTextHeight(
                    descriptionText,
                    doc,
                    width - 32,
                );
                if (currentHeight + textHeight + 8 > height) {
                    newPage();
                }
                doc.text(
                    descriptionText,
                    16,
                    currentHeight,
                    {
                        maxWidth: width - 32,
                    },
                );
                currentHeight += textHeight + 8;
            }

            if (includeWaypoints && $trail.expand?.waypoints_via_trail) {
                const header = $_("waypoints", { values: { n: 2 } });
                let textHeight = getTextHeight(header, doc, width - 32);
                if (currentHeight + textHeight + 8 > height) {
                    newPage();
                }
                doc.setFont("IBMPlexSans-SemiBold", "bold");
                doc.text(header, 16, currentHeight);
                currentHeight += textHeight + 8;

                ($trail.expand.waypoints_via_trail || []).forEach(
                    (waypoint) => {
                        let description = waypoint.description || "";
                        let name = waypoint.name || "";

                        let textHeight =
                            getTextHeight(
                                `${name ? name + "\n" : ""}${description}`,
                                doc,
                                width - 32 - 8,
                            ) + (name ? 2 : 0);
                        if (currentHeight + textHeight + 16 > height) {
                            newPage();
                        }

                        doc.setFont("fa-solid-900", "normal");
                        doc.text(
                            String.fromCharCode(
                                faUnicode(waypoint.icon || "circle"),
                            ),
                            16,
                            currentHeight,
                        );

                        if (name) {
                            doc.setFont("IBMPlexSans-SemiBold", "bold");
                            doc.text(formatHTMLAsText(name), 24, currentHeight);
                            currentHeight +=
                                getTextHeight(
                                    formatHTMLAsText(name),
                                    doc,
                                    width - 32,
                                ) + 2;
                        }

                        doc.setFont("IBMPlexSans-Regular", "normal");
                        doc.text(
                            formatHTMLAsText(description),
                            24,
                            currentHeight,
                            {
                                maxWidth: width - 32 - 8,
                            },
                        );
                        currentHeight +=
                            getTextHeight(
                                formatHTMLAsText(description),
                                doc,
                                width - 32 - 8,
                            ) + 8;
                    },
                );
            }

            addLogo();

            doc.save($trail.name + ".pdf");
            printLoading = false;
        } catch (e) {
            console.error(e);
            show_toast({
                icon: "close",
                type: "error",
                text: $_("error-printing-map"),
            });
            printLoading = false;
        }
    }

    async function updateMapSize() {
        if (!map) {
            return;
        }
        await tick();
        updateScale(map);
    }

    async function updateScale(map: M.Map) {
        const svg = document.getElementById("ruler") as HTMLElement;
        if (!svg) {
            return;
        }
        const svgRect = svg.getBoundingClientRect();
        const height = svgRect.height;
        const width = svgRect.width;

        svg.replaceChildren();

        let oneUnitInPixels = calculatePixelPerMeter(
            map,
            settings && settings.unit == "imperial" ? 1609.34 : 1000,
        );
        let unitsInRuler = width / oneUnitInPixels;

        function createRulerSegment(
            x: number,
            width: number,
            label: string,
            fill: string,
        ) {
            const bar = createRect(x, height / 2, width, 6, fill, "#000");
            const text = createText(
                label,
                x + width - 5 * label.length,
                height / 2 + 20,
                "0.75rem",
            );
            svg.appendChild(bar);
            svg.append(text);
        }
        let multiplier = 1;

        if (unitsInRuler > 500) {
            multiplier = 250;
        } else if (unitsInRuler > 150) {
            multiplier = 50;
        } else if (unitsInRuler > 25) {
            multiplier = 10;
        } else if (unitsInRuler > 15) {
            multiplier = 5;
        } else if (unitsInRuler > 5) {
            multiplier = 2;
        } else if (unitsInRuler > 1.5) {
            multiplier = 1;
        } else if (unitsInRuler > 1) {
            multiplier = 0.5;
        } else if (unitsInRuler > 0.2) {
            multiplier = 0.1;
        } else if (unitsInRuler > 0) {
            multiplier = 0.05;
        }
        const segmentCount = Math.floor(unitsInRuler / multiplier);

        svg.appendChild(createText("0", 0, height / 2 + 20, "0.75rem"));
        for (let i = 0; i < segmentCount; i++) {
            createRulerSegment(
                oneUnitInPixels * i * multiplier,
                oneUnitInPixels * multiplier,
                `${multiplier < 1 ? ((i + 1) * multiplier).toFixed(1) : (i + 1) * multiplier}`,
                i % 2 == 0 ? "#fff" : "#5d5d5d",
            );
        }

        svg.appendChild(
            createText(
                settings && settings.unit == "imperial" ? "MI" : "KM",
                segmentCount * multiplier * oneUnitInPixels - 20,
                height / 2 - 7,
            ),
        );

        scale = calculateScaleFactor(map);
    }

    function toggleGrid() {
        if (selectedGrid == "off") {
            showGrid = false;
        } else {
            showGrid = true;
        }
    }
</script>

<svelte:head>
    <title>{$_("print")} | {page.data.appTitle}</title>
</svelte:head>

<main
    class="grid grid-cols-1 md:grid-cols-[400px_1fr] gap-x-1 gap-y-4 items-start"
>
    <div
        class="print-details md:sticky w-80 md:top-8 mx-auto md:ml-8 p-6 flex flex-col gap-y-4 border border-input-border rounded-3xl"
    >
        <h1 class="text-4xl font-bold">Print Trail</h1>
        <Select
            bind:value={selectedPaperSize}
            items={paperSizes}
            label={$_("paper-size")}
            onchange={updateMapSize}
        ></Select>
        <Select
            bind:value={selectedOrientation}
            items={orientations}
            label={$_("orientation")}
            onchange={updateMapSize}
        ></Select>
        <Select
            bind:value={selectedGrid}
            items={gridOptions}
            label={$_("grid")}
            onchange={toggleGrid}
        ></Select>
        <div>
            <input
                id="description-checkbox"
                type="checkbox"
                bind:checked={includeDescription}
                class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
            />
            <label for="description-checkbox" class="ms-2 text-sm"
                >{$_("include-description")}</label
            >
        </div>
        <div>
            <input
                id="waypoints-checkbox"
                type="checkbox"
                bind:checked={includeWaypoints}
                class="w-4 h-4 bg-input-background accent-primary border-input-border focus:ring-input-ring focus:ring-2"
            />
            <label for="waypoints-checkbox" class="ms-2 text-sm"
                >{$_("include-waypoints")}</label
            >
        </div>

        <Button
            loading={printLoading}
            extraClasses="mt-2"
            primary={true}
            onclick={print}>{$_("print")}!</Button
        >
    </div>

    <div class="paper-container overflow-scroll">
        <div
            id="paper"
            class="flex flex-col mx-auto shadow-xl mb-8 p-8 bg-white"
            style="width: {paperDimensions[selectedPaperSize].css[
                selectedOrientation == 'portrait' ? 'width' : 'height'
            ]}; height: {paperDimensions[selectedPaperSize].css[
                selectedOrientation == 'portrait' ? 'height' : 'width'
            ]}"
        >
            <div class="basis-full">
                <MapWithElevationMaplibre
                    trails={[$trail]}
                    waypoints={$trail.expand?.waypoints_via_trail}
                    activeTrail={0}
                    onzoom={updateScale}
                    bind:map
                    {showGrid}
                    elevationProfileContainer="elevation-profile"
                    mapOptions={{ preserveDrawingBuffer: true }}
                ></MapWithElevationMaplibre>
            </div>
            <div class="flex items-center min-w-0">
                <img class="w-[100px] h-[100px]" id="qrcode" alt="QR Code" />
                <div>
                    <div
                        class="flex mt-1 gap-x-4 gap-y-2 text-sm text-gray-500 whitespace-nowrap"
                    >
                        <span
                            ><i class="fa fa-left-right mr-2"
                            ></i>{formatDistance($trail.distance)}</span
                        >
                        <span
                            ><i class="fa fa-clock mr-2"></i>{formatTimeHHMM(
                                $trail.duration,
                            )}</span
                        >
                        <span
                            ><i class="fa fa-arrow-trend-up mr-2"
                            ></i>{formatElevation($trail.elevation_gain)}</span
                        >
                        <span
                            ><i class="fa fa-arrow-trend-down mr-2"
                            ></i>{formatElevation($trail.elevation_loss)}</span
                        >
                    </div>
                    <svg
                        width="100%"
                        height="50"
                        id="ruler"
                        xmlns="http://www.w3.org/2000/svg"
                        version="1.1"
                    >
                    </svg>
                    <span class="text-sm"
                        >Scale: &nbsp 1 : {Math.round(scale)}</span
                    >
                </div>
                <div
                    class="min-w-0 basis-full relative text-xs"
                    id="elevation-profile"
                ></div>
            </div>
            <div class="pt-4 border-t border-t-input-border z-10">
                <div class="float-right"><LogoText height={48}></LogoText></div>

                <h4 class="text-sm font-semibold">{$trail.name}</h4>
                <h5 class="text-sm">
                    <i class="fa fa-location-dot mr-2"></i>{$trail.location ||
                        "-"}
                </h5>
            </div>
        </div>
    </div>
</main>
