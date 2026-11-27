import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import http from 'node:http';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {createHash} from 'node:crypto';
const exec=promisify(execFile);
const root=await fs.mkdtemp(path.join(os.tmpdir(),'accoding-npm-probe-'));
const npm='npm';
const cli=process.env.npm_execpath || (process.platform==='win32' ? path.join(path.dirname(process.execPath),'node_modules','npm','bin','npm-cli.js') : undefined);
const runNpm=(args,opts={})=>exec(cli?process.execPath:npm,cli?[cli,...args]:args,opts);
const versions=new Map();
const tags=new Map();
const tarballs=new Map();
const available=new Set();
const server=http.createServer((req,res)=>{
  const url=decodeURIComponent(req.url.split('?')[0]);
  if(tarballs.has(url)){res.end(tarballs.get(url));return;}
  const name=url.slice(1);
  const records=versions.get(name);
  if(!records||!available.has(name)){res.writeHead(404);res.end('{}');return;}
  const visible=Object.fromEntries(Object.entries(records).filter(([v])=>available.has(name+'@'+v)));
  res.setHeader('Content-Type','application/json');res.end(JSON.stringify({name,'dist-tags':tags.get(name),versions:visible}));
});
await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
const registry='http://127.0.0.1:'+server.address().port;
try {
  async function add(name,version,dependencies={},optionalDependencies={},tag='latest',extra={}){
    const pkg={name,version,dependencies,optionalDependencies,...extra};
    const dir=await fs.mkdtemp(path.join(root,'pkg-'));
    await fs.writeFile(path.join(dir,'package.json'),JSON.stringify(pkg));
    await fs.writeFile(path.join(dir,'index.js'),'module.exports=1;\n');
    const {stdout}=await runNpm(['pack','--ignore-scripts','--json'],{cwd:dir});
    const packed=JSON.parse(stdout)[0];const bytes=await fs.readFile(path.join(dir,packed.filename));
    const route='/'+name+'/-/'+packed.filename;
    tarballs.set(route,bytes);
    const record={...pkg,dist:{tarball:registry+route,integrity:'sha512-'+createHash('sha512').update(bytes).digest('base64'),shasum:createHash('sha1').update(bytes).digest('hex')}};
    const old=versions.get(name)||{};old[version]=record;versions.set(name,old);
    const t=tags.get(name)||{};t[tag]=version;tags.set(name,t);available.add(name);available.add(name+'@'+version);
  }
  const app='@accoding-probe/app',resource='@accoding-probe/resource',observer='@accoding-probe/observer',missing='@accoding-probe/missing';
  await add(resource,'0.6.9',{}, {},'latest',{scripts:{install:'node -e "require(\'fs\').writeFileSync(\'unexpected-install-script\',\'ran\')"'}});await add(observer,'0.1.0-beta.1',{}, {},'next');
  await add(app,'0.2.8',{[resource]:'^0.6.9'},{[observer]:'0.1.0-beta.1',[missing]:'1.0.0'});
  const install=path.join(root,'install');await fs.mkdir(install);
  await fs.writeFile(path.join(install,'package.json'),JSON.stringify({name:'managed',private:true,dependencies:{[app]:'0.2.8'}}));
  const opts={cwd:install,env:{...process.env,npm_config_registry:registry,npm_config_cache:path.join(root,'cache'),npm_config_fetch_retries:'0',npm_config_audit:'false',npm_config_fund:'false',npm_config_ignore_scripts:'true',npm_config_prefer_online:'true'}};
  const run=async(...args)=>{const r=await runNpm(args,opts);console.log('npm',args.join(' '),r.stdout.trim());return r;};
  const lock=async()=>JSON.parse(await fs.readFile(path.join(install,'package-lock.json'),'utf8'));
  const ver=async(name)=>(await lock()).packages['node_modules/'+name]?.version;
  await run('install');
  assert.equal(await ver(resource),'0.6.9');assert.equal(await ver(observer),'0.1.0-beta.1');
  await assert.rejects(fs.access(path.join(install,'node_modules',missing,'package.json')), {code:'ENOENT'});
  await assert.rejects(fs.access(path.join(install,'node_modules',resource,'unexpected-install-script')), {code:'ENOENT'});
  await add(resource,'0.6.10');await add(resource,'0.7.0');await add(observer,'0.1.0-beta.2',{}, {},'next');
  await run('update','--save');
  assert.equal(await ver(app),'0.2.8');assert.equal(await ver(resource),'0.6.10');assert.equal(await ver(observer),'0.1.0-beta.1');
  assert.equal(JSON.parse(await fs.readFile(path.join(install,'node_modules',resource,'package.json'),'utf8')).version,'0.6.10');
  await add(app,'0.2.9-beta.1',{[resource]:'^0.6.9'},{[observer]:'0.1.0-beta.2'},'next');
  await fs.writeFile(path.join(install,'package.json'),JSON.stringify({name:'managed',private:true,dependencies:{[app]:'0.2.9-beta.1'}}));
  await run('install');await run('update','--save');
  assert.equal(await ver(app),'0.2.9-beta.1');assert.equal(await ver(observer),'0.1.0-beta.2');
  console.log('PASS: exact root and preview dependency, indirect compatible update, incompatible latest rejected, optional missing, staged install + update');
} finally {server.close();await fs.rm(root,{recursive:true,force:true});}
