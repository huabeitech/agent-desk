$headers = @{
    "Authorization" = "Bearer 34bb877b-4ae2-4950-87e7-7282e198ab73"
    "Content-Type"  = "application/json"
}

$authorIdx = "follower_3k2noqx3" # JOY

# Statuses:
# status_xz33599z : Under consideration
# status_p49jn3p1 : Planned
# status_242mo97z : In Development
# status_p47lj9oz : Shipped

# Topics:
# topic_63pxlq1v : Integrations 🔗
# topic_5d9eyzp3 : Improvement 👍

$backlog = @(
    @{
        name = "Telegram Bot Channel Integration"
        description = "Native bidirectional integration with Telegram Bot API. Supports automated zero-config webhook binding, customer conversation routing, AI Agent auto-reply, and human agent outbox delivery."
        status_idx = "status_p47lj9oz" # Shipped
        topic_idxs = @("topic_63pxlq1v")
    },
    @{
        name = "Zalo Official Account (OA) Channel Gateway"
        description = "Native channel adapter for Zalo Official Account (OA). Enables Vietnamese businesses to receive customer support inquiries and dispatch AI/agent replies via Zalo CS messaging API."
        status_idx = "status_p49jn3p1" # Planned
        topic_idxs = @("topic_63pxlq1v")
    },
    @{
        name = "Inbound Email-to-Ticket & SMTP/IMAP Gateway"
        description = "Convert inbound customer support emails into threaded conversation tickets automatically. Allows agents and AI to reply directly via email."
        status_idx = "status_p49jn3p1" # Planned
        topic_idxs = @("topic_63pxlq1v", "topic_5d9eyzp3")
    },
    @{
        name = "WhatsApp Business API & Cloud Gateway"
        description = "Connect WhatsApp Business Cloud API to Crove Desk. Support template messages, interactive buttons, and real-time chat sync for international customer support."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_63pxlq1v")
    },
    @{
        name = "Live Chat Web Widget SDK with Custom Theming & JWT Verification"
        description = "Embeddable lightweight web chat widget with customizable theme colors, position, and secure customer JWT token verification."
        status_idx = "status_p47lj9oz" # Shipped
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "OpenAI-Compatible AI Engine & Auto-Bootstrap"
        description = "Zero-config LLM and vector embedding integration supporting OpenAI, DOS.AI, DeepSeek, and OpenAI-compatible gateways via environment variables."
        status_idx = "status_p47lj9oz" # Shipped
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Smart Answerability Gate & Confidence Scoring for RAG"
        description = "Evaluates retrieval confidence and document relevancy before AI generates a response, preventing hallucinations on unsupported customer questions."
        status_idx = "status_p47lj9oz" # Shipped
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Automated Human Handoff on Low AI Confidence"
        description = "Seamlessly escalates customer conversations to online human support agents with context transfer when the AI Answerability Gate confidence is below threshold."
        status_idx = "status_p49jn3p1" # Planned
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Visual AI Workflow Canvas & Node-based Orchestration"
        description = "Legacy node-based drag-and-drop workflow designer (Flowgram) for deterministic multi-step support flows. (Kept under consideration in favor of dynamic AI-native agentic loops)."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Automated Conversation Summarization & Sentiment Analysis"
        description = "AI automatically generates resolution summaries and tags customer sentiment (Positive, Neutral, Frustrated) upon ticket closure."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "2-Tier Hybrid Sync: Relational Mirror with Twenty CRM & DOS.Me"
        description = "Real-time bidirectional synchronization of Company and Customer profiles between Twenty CRM, DOS.Me, and Crove Desk via webhook events."
        status_idx = "status_p47lj9oz" # Shipped
        topic_idxs = @("topic_63pxlq1v", "topic_4d2x1y1v")
    },
    @{
        name = "MCP Tool Calling: Live Deal & Subscription Status Lookup from CRM"
        description = "Equips Crove Desk AI Agents with Model Context Protocol (MCP) tools to query live CRM deals, subscription tiers, and customer records on demand."
        status_idx = "status_p49jn3p1" # Planned
        topic_idxs = @("topic_63pxlq1v", "topic_4d2x1y1v")
    },
    @{
        name = "Auto-Create CRM Deals & Follow-up Tasks from Support Inquiries"
        description = "AI Agent identifies sales opportunities during customer support conversations and automatically creates Deals and follow-up Tasks in Twenty CRM."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_63pxlq1v", "topic_4d2x1y1v")
    },
    @{
        name = "Multi-Tenant Workspace Management with Just-In-Time SSO"
        description = "Isolated multi-organization workspace switching, member role management, and JIT user provisioning via DOS.Me OIDC single sign-on."
        status_idx = "status_p47lj9oz" # Shipped
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Configurable SLA Policies & Priority Escalation Rules"
        description = "Define First Response Time and Resolution Time SLA targets based on customer tier, ticket priority, and business hours with automated alerts."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Granular Role-Based Access Control (RBAC) for Support Agents"
        description = "Customizable permission matrices for Tier 1 agents, senior support specialists, and support administrators across channels and knowledge bases."
        status_idx = "status_p49jn3p1" # Planned
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Multi-language Knowledge Base & Vector FAQ Indexing"
        description = "Publish help documentation and categorized FAQs with multilingual support (EN, VI, ZH) and automatic Qdrant vector embedding indexing."
        status_idx = "status_p47lj9oz" # Shipped
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Public Customer Community Forum & Peer Discussion Board"
        description = "Community discussion space allowing customers to post questions, share tips, vote on best answers, with agent moderation."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Custom Domain & White-Label Support Portal"
        description = "CNAME custom domain mapping and custom branding (colors, logos, favicons) for customer-facing Help Centers."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_5d9eyzp3")
    },
    @{
        name = "Omnichannel CSAT & Customer Satisfaction Surveys"
        description = "Trigger automated CSAT star ratings and feedback prompts across Web Widget, Telegram, and Zalo OA when tickets are resolved."
        status_idx = "status_xz33599z" # Under consideration
        topic_idxs = @("topic_5d9eyzp3")
    }
)

foreach ($item in $backlog) {
    Write-Host "Creating idea: $($item.name) [Status: $($item.status_idx)]..."
    $body = @{
        name = $item.name
        description = $item.description
        status_idx = $item.status_idx
        topic_idxs = $item.topic_idxs
        author_idx = $authorIdx
    } | ConvertTo-Json -Depth 5 -Compress

    $success = $false
    for ($attempt = 1; $attempt -le 4; $attempt++) {
        try {
            $resp = Invoke-RestMethod -Uri "https://api.frill.co/v1/ideas" -Method Post -Headers $headers -Body $body
            Write-Host "  -> Success! Idx: $($resp.data.idx) Slug: $($resp.data.slug)"
            $success = $true
            break
        } catch {
            Write-Host "  -> Attempt $attempt failed: $_"
            if ($_.Exception.Response) {
                $stream = $_.Exception.Response.GetResponseStream()
                $reader = New-Object System.IO.StreamReader($stream)
                $errBody = $reader.ReadToEnd()
                Write-Host "  -> Response: $errBody"
                if ($errBody -match "rate limit") {
                    Write-Host "  -> Sleeping 60s for rate limit reset..."
                    Start-Sleep -Seconds 60
                }
            }
            Start-Sleep -Seconds 3
        }
    }
    Start-Sleep -Seconds 2
}
