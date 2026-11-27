import {test} from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import {execFile} from "node:child_process";
import {promisify} from "node:util";
import {fileURLToPath} from "node:url";

test("launcher guide renders identically from LF and CRLF checkouts", async () => {
  const root = fileURLToPath(new URL("../../", import.meta.url));
  const tmp = await fs.mkdtemp(path.join(os.tmpdir(), "routilet-guide-"));
  try {
    const source = (await fs.readFile(path.join(root, "docs/launcher/使用说明.html"), "utf8")).replace(/\r\n/g, "\n");
    const outputs = [];
    for (const [name, content] of [["lf", source], ["crlf", source.replace(/\n/g, "\r\n")]]) {
      const input = path.join(tmp, name + ".html");
      const output = path.join(tmp, name + "-rendered.html");
      await fs.writeFile(input, content);
      await promisify(execFile)(process.execPath, [path.join(root, "scripts/render-launcher-guide.mjs"), input, output, "0.2.8", "0.3.0", "1"]);
      outputs.push(await fs.readFile(output, "utf8"));
    }
    assert.equal(outputs[0], outputs[1]);
    assert.ok(outputs[0].includes("Accoding Companion 0.2.8"));
    assert.ok(outputs[0].includes("回退、切换渠道及卸载"));
  } finally {
    await fs.rm(tmp, {recursive: true, force: true});
  }
});
