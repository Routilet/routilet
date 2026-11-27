import {test} from "node:test";
import {execFile} from "node:child_process";
import {promisify} from "node:util";
import path from "node:path";

test("npm resolves compatible transitive updates and installs reviewed lock", {timeout:180000}, async()=>{
  const {stdout}=await promisify(execFile)(process.execPath,[path.resolve("tests/standalone/npm-dependency-probe.mjs")],{timeout:170000,maxBuffer:1024*1024});
  if(!stdout.includes("PASS: exact root")) throw Error(stdout);
});
