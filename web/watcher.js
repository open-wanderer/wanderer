import fs from 'node:fs';
import path from 'node:path';
import chokidar from 'chokidar';

const ORIGIN = "http://localhost:3000";
const { UPLOAD_FOLDER } = process.env;

if (UPLOAD_FOLDER) {
  const uploadPath = path.resolve(UPLOAD_FOLDER);

  // Ensure directory exists
  if (!fs.existsSync(uploadPath)) {
    try {
      fs.mkdirSync(uploadPath, { recursive: true });
    } catch (err) {
      console.error(`[File Watcher] Failed to create directory ${uploadPath}:`, err.message);
    }
  }

  console.log(`[File Watcher] Service active. Watching: ${uploadPath}`);

  const watcher = chokidar.watch(uploadPath, {
    persistent: true,
    ignoreInitial: true,
    awaitWriteFinish: {
      stabilityThreshold: 2000,
      pollInterval: 100
    }
  });

  watcher.on('add', async (filePath) => {
    const relative = path.relative(uploadPath, filePath);
    const [token] = relative.split(path.sep);

    if (!token || token === '.' || path.basename(filePath).startsWith('.')) return;

    try {
      const fileBuffer = fs.readFileSync(filePath);
      const fileBlob = new Blob([fileBuffer]);
      const formData = new FormData();
      formData.append('file', fileBlob, path.basename(filePath));
      formData.append('ignoreDuplicates', "true");

      const response = await fetch(`${ORIGIN}/api/v1/trail/upload`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}` },
        body: formData
      });

      if (response.ok) {
        fs.unlinkSync(filePath);
        console.log(`[File Watcher] Uploaded and removed: ${path.basename(filePath)}`);
      } else {
        const errorText = await response.text();
        console.error(`[File Watcher] Server rejected ${path.basename(filePath)} (${response.status}): ${errorText}`);
      }
    } catch (err) {
      console.error(`[File Watcher] Upload error:`, err.message);
    }
  });

  watcher.on('error', error => console.error(`[File Watcher] Watcher error: ${error}`));

  process.on('SIGTERM', () => {
    watcher.close();
  });

} else {
  console.log('[File Watcher] Disabled: UPLOAD_FOLDER not provided.');
}