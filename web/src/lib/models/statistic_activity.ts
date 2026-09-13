import type { SummitLog } from "./summit_log";

export type StatisticActivitySource = "summit_log" | "completed_trail";

export interface StatisticActivity extends SummitLog {
    source: StatisticActivitySource;
    collectionId: string;
    collectionName: string;
}
