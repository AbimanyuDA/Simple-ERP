/* Render local documentation artifacts only; no application/server/database mutation. */
const fs = require("node:fs/promises");
const path = require("node:path");
const root = path.resolve(__dirname, "../../..");
const out = path.resolve(__dirname, "..");
const { chromium } = require(path.join(root, "frontend/node_modules/playwright"));

async function main() {
  const manifest = JSON.parse(await fs.readFile(path.join(out, "manifest.json"), "utf8"));
  const qa = path.join(out, "qa");
  await fs.mkdir(qa, { recursive: true });
  const browser = await chromium.launch({ headless: true,
    ...(process.env.BPMN_CHROMIUM_PATH ? { executablePath: process.env.BPMN_CHROMIUM_PATH } : {}) });
  const page = await browser.newPage({ viewport: { width: 1500, height: 1000 }, deviceScaleFactor: 1 });
  const selected = new Set(["l0-simple-erp", "l1-p2p", "l2-ap", "l2-po-gr", "l3-so-review",
    "l3-ap-payment-post", "role-staff", "l2-pr", "l2-so", "l2-adjustment", "l2-delivery",
    "l2-ar", "l3-kontrol-request", "role-supervisor", "role-super-admin", "l2-p2p-correction",
    "l2-purchase-correction", "l3-pr-review", "l3-ap-match"]);
  const errors = [];
  const perModel = [];
  let rendered = 0;
  try {
    for (const model of manifest.models) {
      const svg = await fs.readFile(path.join(out, "preview", model.id + ".svg"), "utf8");
      await page.setContent('<!doctype html><html><body style="margin:0">' + svg + "</body></html>");
      const check = await page.evaluate(() => {
        const svg = document.querySelector("svg");
        if (!svg) return { error: "SVG missing" };
        const box = svg.viewBox.baseVal;
        const texts = [...svg.querySelectorAll("text")];
        const clipped = texts.filter(node => {
          const b = node.getBBox();
          return b.x < 0 || b.y < 0 || b.x+b.width > box.width || b.y+b.height > box.height;
        }).map(node => node.textContent);
        const collisions = [];
        for (let i = 0; i < texts.length; i++) {
          const a = texts[i].getBBox();
          for (let j = i+1; j < texts.length; j++) {
            const b = texts[j].getBBox();
            if (Math.min(a.x+a.width, b.x+b.width)-Math.max(a.x,b.x) > 2 &&
                Math.min(a.y+a.height, b.y+b.height)-Math.max(a.y,b.y) > 2) {
              collisions.push([texts[i].textContent, texts[j].textContent]);
            }
          }
        }
        return { clipped, collisions };
      });
      if (check.error || check.clipped.length || check.collisions.length) errors.push({ model: model.id, ...check });
      perModel.push({ model: model.id, level: model.level, passed: !check.error &&
        check.clipped.length === 0 && check.collisions.length === 0,
        clipped: check.clipped.length, text_collisions: check.collisions.length });
      if (selected.has(model.id)) {
        await page.screenshot({ path: path.join(qa, model.id + ".png"), fullPage: true });
      }
      rendered++;
      if (rendered % 20 === 0) console.log(`Rendered ${rendered}/${manifest.models.length}`);
    }
  } finally {
    await browser.close();
  }
  const report = { rendered, screenshot_samples: [...selected], passed: errors.length === 0,
    checks: ["Chromium SVG load", "text within SVG viewport", "text-to-text collisions"],
    limits: "Rendering checks do not prove process semantics or detect every edge/label collision.",
    per_model: perModel, errors };
  await fs.writeFile(path.join(qa, "render-validation.json"), JSON.stringify(report, null, 2) + "\n");
  console.log(JSON.stringify(report, null, 2));
  if (errors.length) process.exitCode = 1;
}
main().catch(error => { console.error(error); process.exitCode = 1; });
