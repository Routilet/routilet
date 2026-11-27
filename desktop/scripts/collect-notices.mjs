import fs from "node:fs/promises";
import path from "node:path";
import { execFileSync } from "node:child_process";

// The shipped native binaries statically contain Go and Go dependencies.
// Preserve upstream license and notice texts, including module subdirectories.
const legalName = /^(?:licen[sc]es?|copying|copyright|notice|patents|unlicense)(?:$|[._-])/i;
async function legalFiles(root, recursive = true) {
  const results = [];
  async function visit(dir) {
    for (const entry of await fs.readdir(dir, { withFileTypes: true })) {
      const file = path.join(dir, entry.name);
      if (entry.isDirectory() && recursive && entry.name !== ".git") await visit(file);
      else if (entry.isFile() && legalName.test(entry.name)) results.push(file);
    }
  }
  await visit(root);
  return results.sort();
}

export async function collectNotices(destination, targets, entry='./cmd/c-observer') {
  if (!targets?.length) throw Error("No distribution targets specified");
  // The full module graph includes test-only and platform-exclusive dependencies.
  // Collect the union of packages required by the actual shipped builds instead.
  // A newly shipped target is automatically included and missing licenses still fail.
  const modules = new Map();
  for (const target of targets) {
    const lines = execFileSync("go", ["list", "-deps", "-tags", "production", "-f", "{{with .Module}}{{if not .Main}}{{.Path}}\\t{{.Version}}\\t{{.Dir}}{{end}}{{end}}", entry], {
      encoding: "utf8", env: { ...process.env, CGO_ENABLED: "0", ...target }
    });
    for (const line of lines.split("\n").filter(Boolean)) {
      const [name, version, dir] = line.split("\\t");
      if (!name || !version || !dir) throw Error("Incomplete module information: " + line);
      modules.set(name + " " + version, dir);
    }
  }
  const sections = [
    "# Third-party licenses and notices",
    "",
    "This distribution includes native Go binaries. The following upstream legal texts are retained from the Go toolchain and the union of Go package dependencies of the shipped build targets. pnpm license texts are distributed separately with the initial launcher.",
    ""
  ];
  async function append(label, root, recursive) {
    if (!root) throw Error("Missing module directory for " + label);
    const files = await legalFiles(root, recursive);
    if (!files.some(file => /^(?:licen[sc]es?|copying|unlicense)(?:$|[._-])/i.test(path.basename(file)))) {
      // Upstream v0.0.1 explicitly declares MIT in its README but ships no LICENSE.
      // Preserve the actual upstream declaration plus standard MIT terms; never
      // extend this exception to a different module or version without review.
      if (label !== "github.com/mattn/go-localereader v0.0.1")
        throw Error("No upstream license found for " + label + "; review before redistribution");
      const readme = await fs.readFile(path.join(root, "README.md"), "utf8");
      if (!/^## License\s+MIT\s*$/m.test(readme) || !readme.includes("Yasuhiro Matsumoto"))
        throw Error("Reviewed upstream license declaration changed: " + label);
      sections.push("## " + label, "", "--- upstream README.md ---", readme.trimEnd(), "",
        "--- Standard MIT license terms (upstream explicitly declares MIT; no separate copyright year supplied) ---",
        (await fs.readFile(new URL("../licenses/go-localereader-MIT.txt", import.meta.url), "utf8")).trimEnd(), "");
    } else {
      sections.push("## " + label, "");
    }
    for (const file of files) {
      sections.push("--- " + path.relative(root, file).split(path.sep).join("/") + " ---", (await fs.readFile(file, "utf8")).trimEnd(), "");
    }
  }
  const goroot = execFileSync("go", ["env", "GOROOT"], { encoding: "utf8" }).trim();
  const goversion = execFileSync("go", ["version"], { encoding: "utf8" }).trim();
  await append(goversion, goroot, false);
  for (const [label, dir] of [...modules].sort(([a],[b]) => a.localeCompare(b))) {
    await append(label, dir, true);
  }
  await fs.writeFile(destination, sections.join("\n") + "\n");
}
