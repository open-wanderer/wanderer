export type BackgroundTaskStatus =
    | "enqueued"
    | "running"
    | "success"
    | "warning"
    | "error"
    | "cancelled";

export type BackgroundTaskAction = {
    label: string;
    icon: string;
    run?: () => void | Promise<void>;
    href?: string;
};

export type BackgroundTask = {
    id: string;
    title: string;
    detail?: string;
    status: BackgroundTaskStatus;
    progress?: number;
    actions?: BackgroundTaskAction[];
    onDismiss?: () => void;
};

class BackgroundTaskStore {
    tasks: BackgroundTask[] = $state([]);
}

export const backgroundTaskStore = new BackgroundTaskStore();

export function createBackgroundTaskId(prefix: string = "task") {
    const random =
        typeof crypto !== "undefined" && "randomUUID" in crypto
            ? crypto.randomUUID()
            : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
    return `${prefix}-${random}`;
}

export function upsertBackgroundTask(task: BackgroundTask) {
    const index = backgroundTaskStore.tasks.findIndex((t) => t.id === task.id);
    if (index >= 0) {
        backgroundTaskStore.tasks[index] = task;
        return;
    }
    backgroundTaskStore.tasks.push(task);
}

export function removeBackgroundTask(id: string) {
    const index = backgroundTaskStore.tasks.findIndex((t) => t.id === id);
    if (index >= 0) {
        backgroundTaskStore.tasks.splice(index, 1);
    }
}
