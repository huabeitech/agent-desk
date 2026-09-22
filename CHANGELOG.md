# Changelog

All notable changes to the **Crove Desk** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.3.0] - 2026-08-26

### Added
- **2-Tier Hybrid Architecture**: Implemented local database mirroring for `Company` and `Customer` entities alongside deep Agentic Tool Calling via MCP.
- **Bi-directional Webhook Synchronization**: Added support for event-driven synchronization (`company.created`, `company.updated`, `customer.created`, `customer.updated`, `organization.created`, `organization.member.added`) with HMAC-SHA256 signature verification.
- **Twenty CRM MCP Integration**: Added support for connecting Crove Desk AI Agent to Twenty CRM MCP server (`twenty_crm.get_subscription_status`, `twenty_crm.create_opportunity`, `twenty_crm.create_task`).
- **Vietnamese Language Support (`vi-VN`)**: Added complete Vietnamese localization files and implemented `LanguageToggle` component in navigation header and user menu.
- **AI Agent Loop Live Test Suite**: Added comprehensive live integration tests for Answerability Gate, Knowledge Base context grounding, and function calling tool loops.

### Changed
- **Typography & Font Resolution**: Replaced broken Geist/Times New Roman font fallback with Tailwind CSS v4 `@theme inline` mapping to Inter font with Latin and Vietnamese character subsets.
- **Sidebar Font Sizing**: Refined dashboard navigation sidebar font size to compact 13.5px / 13px for improved scannability and professional desktop density.
- **System Architecture Documentation**: Updated `docs/ARCHITECTURE.md` with 2-Tier Hybrid Architecture diagrams and webhook event specifications.

---

## [0.2.0] - 2026-08-25

### Added
- **OpenAI-Compatible AI Configuration**: Supported configuring LLM and Embedding models via `.env` environment variables (`OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_LLM_MODEL`, `OPENAI_EMBEDDING_MODEL`, `OPENAI_EMBEDDING_DIMENSION`).
- **DOS.AI Provider Integration**: Configured live support for DOS.AI (`dos-ai` LLM model and `qwen3-embedding-4b` 2560-dim embedding model).
- **Automated AI Bootstrap & Sync**: Implemented `InitAI` startup hook to automatically seed and synchronize default AI model configurations into PostgreSQL.
- **Default Crove Desk Knowledge Base**: Added auto-seeding of official Crove Desk Knowledge Base and 7 core FAQ entries with background vector indexing in Qdrant.
- **Dynamic Company Branding**: Added `COMPANY_NAME` and `COMPANY_LOGO_URL` configuration exposed via `/api/config` and applied to Login, Workspace Switcher, Legal document pages, and Support Center header.
- **OAuth 2.1 with PKCE S256**: Implemented secure OIDC authorization code exchange with PKCE code challenge and verifier.
- **Password Login Toggle**: Added `PASSWORD_LOGIN_ENABLED` setting to enforce SSO-only login flows.

### Fixed
- Fixed Next.js static export SPA routing for `/dashboard/` trailing slashes.
- Fixed embedded locale file path resolution on Windows environments.

---

## [0.1.0] - 2026-08-22

### Added
- **Repository Initialization**: Forked from `huabeitech/agent-desk` to `DOS/Crove-Desk`.
- **PostgreSQL Database Support**: Added PostgreSQL driver (`gorm.io/driver/postgres`) and normalized GORM model schema types for cross-database compatibility (PostgreSQL, MySQL, SQLite).
- **Supabase Integration**: Connected to Supabase `dos.me` PostgreSQL database under schema `desk` with Session Pooler.
- **Multi-tenant Organization Architecture**: Added `Organization` and `OrganizationMember` models with JIT workspace provisioning upon OIDC login.
- **Production Deployment**: Configured `docker-compose.prod.yml`, Qdrant Vector DB, and Cloudflare Tunnel routing for `desk.crove.com` on GCP VM `crove-server`.
