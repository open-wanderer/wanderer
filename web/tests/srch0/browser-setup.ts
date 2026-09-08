import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { fileURLToPath } from 'node:url';

export default async function setup() {
    const script = fileURLToPath(new URL('../../../scripts/srch0/browser-server.mjs', import.meta.url));
    const child = spawn(process.execPath, [script], { stdio: ['ignore', 'inherit', 'inherit', 'ipc'] });
    const ready = new Promise<string>((resolve, reject) => {
        child.once('message', (message: { baseURL?: string }) => {
            if (message.baseURL) resolve(message.baseURL);
            else reject(new Error('Der Browserserver hat keine URL geliefert.'));
        });
        child.once('error', reject);
        child.once('exit', code => reject(new Error(`Browserserver vorzeitig beendet: ${code}`)));
    });
    const timeout = setTimeout(() => child.kill('SIGTERM'), 120000);
    try {
        process.env.SRCH0_BROWSER_URL = await ready;
    } catch (error) {
        child.kill('SIGTERM');
        throw error;
    } finally {
        clearTimeout(timeout);
    }
    return async () => {
        if (child.exitCode !== null || child.signalCode !== null) return;
        const exited = once(child, 'exit');
        child.send({ stop: true });
        await exited;
    };
}
