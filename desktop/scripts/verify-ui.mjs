import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
const root=new URL('../frontend/',import.meta.url);
const manifest=JSON.parse(await readFile(new URL('ui/manifest.json',root),'utf8'));
for(const [file,sha] of Object.entries(manifest.files)){
 const bytes=await readFile(new URL(file,root));
 const actual=createHash('sha1').update('blob '+bytes.length+'\0').update(bytes).digest('hex');
 if(actual!==sha)throw Error('UI snapshot differs: '+file);
}
console.log('Verified Routilet UI '+manifest.version+' at '+manifest.sourceCommit);
