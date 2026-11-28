import {collectNotices} from './collect-notices.mjs';
await collectNotices(process.argv[2],[{CGO_ENABLED:process.env.CGO_ENABLED}],'.');
import {appendFile,readFile} from 'node:fs/promises';
await appendFile(process.argv[2], '\n\n## Lucide SVG icons (0.456.0)\n'+await readFile('frontend/ui/icons/LICENSE','utf8'));
