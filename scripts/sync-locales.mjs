import fs from "node:fs";
import path from "node:path";

const root = path.resolve(new URL(".", import.meta.url).pathname, "..");

const sourceDir = path.join(root, "web-admin", "src", "locales");
const files = ["zh-CN.json", "en-US.json"];
const targetDirs = [
  path.join(root, "web-admin", "public", "assets", "locales"),
  path.join(root, "web-sdk", "src", "locales"),
  path.join(root, "web-sdk", "public", "locales"),
  path.join(root, "server", "web", "admin", "assets", "locales"),
  path.join(root, "server", "web", "sdk", "locales"),
];

function ensureDir(dir) {
  fs.mkdirSync(dir, { recursive: true });
}

function copyIfChanged(from, to) {
  const content = fs.readFileSync(from);
  if (fs.existsSync(to)) {
    const old = fs.readFileSync(to);
    if (Buffer.compare(content, old) === 0) {
      return false;
    }
  }
  fs.writeFileSync(to, content);
  return true;
}

let changed = 0;
for (const file of files) {
  const from = path.join(sourceDir, file);
  if (!fs.existsSync(from)) {
    throw new Error(`missing source locale file: ${from}`);
  }
  for (const dir of targetDirs) {
    ensureDir(dir);
    const to = path.join(dir, file);
    if (copyIfChanged(from, to)) changed += 1;
  }
}

console.log(`[sync-locales] done, changed files: ${changed}`);
