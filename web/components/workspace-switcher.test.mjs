import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { describe, it } from "node:test";

const source = await readFile(new URL("./workspace-switcher.tsx", import.meta.url), "utf8");
const appSidebarSource = await readFile(new URL("./app-sidebar.tsx", import.meta.url), "utf8");

describe("workspace switcher config", () => {
  it("contains dashboard and workbench destinations", async () => {
  assert.match(source, /key:\s*"dashboard"[\s\S]*href:\s*"\/dashboard"[\s\S]*labelKey:\s*"workspace\.dashboard"/);
  assert.match(source, /key:\s*"workbench"[\s\S]*href:\s*"\/workbench"[\s\S]*labelKey:\s*"workspace\.workbench"/);
});

it("wraps the dropdown label in a Base UI menu group", async () => {
  assert.match(source, /<DropdownMenuGroup>[\s\S]*<DropdownMenuLabel>\{t\("workspace\.switchWorkspace"\)\}<\/DropdownMenuLabel>/);
});

it("does not auto-open the rail menu from focus or hover events", async () => {
  assert.doesNotMatch(source, /onFocus=\{openHoverMenu\}/);
  assert.doesNotMatch(source, /onBlur=\{closeHoverMenu\}/);
  assert.doesNotMatch(source, /onPointerEnter=\{openHoverMenu\}/);
  assert.doesNotMatch(source, /onPointerLeave=\{closeHoverMenu\}/);
  assert.doesNotMatch(source, /hoverOpen/);
});

it("centers the dashboard switcher logo and shows a collapsed switch indicator", async () => {
  assert.match(source, /variant === "sidebar" &&[\s\S]*group-data-\[collapsible=icon\]:p-0!/);
  assert.match(source, /variant === "sidebar" &&[\s\S]*group-data-\[collapsible=icon\]:justify-center/);
  assert.match(source, /const switchIndicatorClassName =\s*"absolute bottom-0\.5 right-0\.5 size-2\.5/);
  assert.match(source, /variant === "rail" \? \([\s\S]*<ChevronsUpDownIcon className=\{switchIndicatorClassName\} \/>/);
  assert.match(source, /className=\{cn\(switchIndicatorClassName, "hidden group-data-\[collapsible=icon\]:block"\)\}/);
});

it("uses the same compact trigger footprint for dashboard collapsed and workbench rail switchers", async () => {
  assert.match(source, /variant === "rail" &&\s*"relative size-8 rounded-md/);
  assert.doesNotMatch(source, /variant === "rail" &&\s*"relative size-11/);
});
});
