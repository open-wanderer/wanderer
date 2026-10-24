import { uploadBodyLimit } from './src/lib/server/upload_body_limit.js';

// Configure the adapter before it reads its request ceiling. Keep the same
// default and accepted values as the multipart request hook.
process.env.BODY_SIZE_LIMIT = String(uploadBodyLimit(process.env.BODY_SIZE_LIMIT));
await import('./build/index.js');
