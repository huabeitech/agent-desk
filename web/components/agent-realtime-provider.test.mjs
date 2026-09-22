import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";

import { describe, it } from "node:test";

const providerSource = await readFile(
  new URL("./agent-realtime-provider.tsx", import.meta.url),
  "utf8",
).catch(() => "");
const workbenchLayoutSource = await readFile(
  new URL("../app/(dashboard)/workbench/layout.tsx", import.meta.url),
  "utf8",
);
const dashboardConversationsPageSource = await readFile(
  new URL("../app/(dashboard)/dashboard/conversations/page.tsx", import.meta.url),
  "utf8",
);
const conversationWorkbenchSource = await readFile(
  new URL("../app/(dashboard)/dashboard/conversations/_components/conversation-workbench.tsx", import.meta.url),
  "utf8",
);

describe("AgentRealtimeProvider placement", () => {
  it("owns the agent realtime hook", () => {
    assert.match(providerSource, /export function AgentRealtimeProvider/);
    assert.match(providerSource, /useAgentConversationRealtime\(\)/);
  });

  it("is mounted at the workbench layout level", () => {
    assert.match(workbenchLayoutSource, /import \{ AgentRealtimeProvider \}/);
    assert.match(workbenchLayoutSource, /<AgentRealtimeProvider \/>/);
  });

  it("keeps dashboard conversations realtime without tying it to ConversationWorkbench", () => {
    assert.match(dashboardConversationsPageSource, /<AgentRealtimeProvider \/>/);
    assert.doesNotMatch(conversationWorkbenchSource, /useAgentConversationRealtime/);
  });
});
