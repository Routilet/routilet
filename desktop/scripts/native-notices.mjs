import {collectNotices} from './collect-notices.mjs';
await collectNotices(process.argv[2],[{CGO_ENABLED:process.env.CGO_ENABLED}],'.');
