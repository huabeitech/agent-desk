import assert from "node:assert/strict"
import test from "node:test"
import ts from "typescript"
import { readFile } from "node:fs/promises"
import vm from "node:vm"

async function loadModule() {
  const source = await readFile(new URL("./notification-i18n.ts", import.meta.url), "utf8")
  const compiled = ts.transpileModule(source, {
    compilerOptions: {
      target: ts.ScriptTarget.ES2017,
      module: ts.ModuleKind.CommonJS,
    },
    fileName: "notification-i18n.ts",
  })
  const sandbox = {
    exports: {},
    module: { exports: {} },
    require: (id) => {
      if (id === "@/i18n/config") {
        return {
          DEFAULT_LOCALE: "zh-CN",
          normalizeLocale: (loc) => (loc === "en-US" || loc === "en" ? "en-US" : "zh-CN"),
        }
      }
      if (id === "@/i18n/messages") {
        return {
          translateMessage: (loc, key, values) => {
            if (key === "notification.fallbackTitle") return "Notification"
            if (key === "notification.ticketAssignedTitle") return "Ticket assigned"
            if (key === "notification.ticketAssignedLine") return `Ticket ${values?.ticketNo} has been assigned to you.`
            if (key === "notification.assignmentReason") return `Assignment reason: ${values?.reason}`
            if (key === "notification.conversationTransferredTitle") return "Conversation transferred"
            if (key === "notification.conversationAutoAssignedTitle") return "Conversation auto-assigned"
            if (key === "notification.conversationAssignedTitle") return "Conversation assigned"
            if (key === "notification.conversationAssignedLine") return `Conversation #${values?.conversationId} was assigned to you`
            if (key === "notification.transferReason") return `Transfer reason: ${values?.reason}`
            return key
          },
        }
      }
      throw new Error(`unexpected import ${id}`)
    },
  }
  sandbox.exports = sandbox.module.exports
  vm.runInNewContext(compiled.outputText, sandbox)
  return sandbox.module.exports
}

test("localizes realtime ticket assignment notification to English", async () => {
  const { localizeNotificationItem } = await loadModule()

  const result = localizeNotificationItem(
    {
      id: 1,
      recipientUserId: 2,
      title: "\u5de5\u5355\u6307\u6d3e\u63d0\u9192",
      content: "\u5de5\u5355 TK-100 \u5df2\u6307\u6d3e\u7ed9\u4f60\nCannot sign in\n\u6307\u6d3e\u539f\u56e0: urgent",
      notificationType: "ticket_assigned",
      bizType: "ticket",
      bizId: 100,
      actionUrl: "/dashboard/tickets?ticketId=100",
    },
    "en-US"
  )

  assert.equal(result.title, "Ticket assigned")
  assert.equal(
    result.content,
    "Ticket TK-100 has been assigned to you.\nCannot sign in\nAssignment reason: urgent"
  )
})
