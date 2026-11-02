# 🚀 LuciNexus: Autonomous, Resilient AI Agent for OpenWrt

[Release: v1.0.0](https://github.com/Synapse-Works/Luci-Nexus/releases)
[License: Apache 2.0](https://github.com/Synapse-Works/Luci-Nexus/blob/main/LICENSE)
[](#-cross-compilation)
[](#-safety--resilience-architecture)
[](#-embedded-hardware-optimization)

**LuciNexus** is an enterprise-grade, memory-efficient autonomous AI agent built natively for OpenWrt routers. It empowers you to administer, configure, and troubleshoot your router using natural language commands—safely backed by atomic transaction rollbacks, zero-latency offline recovery, and strict typed tool calling.

---

## ✨ Key Features

- 🗣️ **Conversational Router Administration:** Control Wi-Fi, firewall rules, DHCP leases, port forwarding, and system services in plain English (e.g., _"turn on 5GHz Wi-Fi"_, _"open port 8080 for web server"_).
- 🛡️ **Atomic Snapshot & Auto-Rollback:** Automatically snapshots `/etc/config` before applying changes. If post-execution network or syntax health checks fail, the system instantly rolls back to prevent bricking or lost management access.
- ⚡ **Zero-Latency Offline Fastpath:** Built-in deterministic pattern engine executes routine operations (_Wi-Fi reload, WAN DHCP renew, ping diagnostics, uptime_) locally with **<10ms latency**, requiring **zero internet connectivity and zero API tokens**.
- 🔌 **Zero-Click cURL AI Provider Import:** Connect any LLM endpoint (DeepSeek, Groq, OpenRouter, Together AI, or local Ollama/vLLM) by copy-pasting a standard `curl` command.
- 🔒 **Strict Typed Tool Calling (No Shell Injections):** Operates via validated JSON schemas (`uci_get`, `uci_set`, `uci_commit`, `service_control`, `network_diagnose`) rather than arbitrary shell string execution.
- 🪶 **Embedded Hardware Optimization:** Employs intent-driven sparse context hydration to keep resident memory footprint **under 15MB RAM**, making it fully compatible with low-spec MIPS and ARM routers.
- 🖥️ **Modern LuCI Web Interface (`luci-app-lucinexus`):** Reactive JavaScript dashboard with live plan previews, configuration diff inspection, rollback triggers, and cURL auto-detection.

---

## 🏗️ Architecture Overview

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        User Interface Layer                            │
│        (CLI Terminal / LuCI Web UI / Background REST & JSON-RPC)       │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│               Zero-Latency Offline Fastpath Engine                     │
│        Matches common recovery intents locally without internet        │
└─────────────────┬──────────────────────────────────────┬───────────────┘
                  │ (Matched: <10ms)                     │ (Complex Query)
                  ▼                                      ▼
    ┌───────────────────────────┐      ┌─────────────────────────────────┐
    │  Direct Local Execution   │      │    Sparse Context Hydration     │
    └───────────────────────────┘      │ Queries only target subsystem   │
                                       │ (<15MB RAM resident footprint)  │
                                       └────────────────┬────────────────┘
                                                        │
                                                        ▼
                                       ┌─────────────────────────────────┐
                                       │   LLM Structured Tool Engine    │
                                       │ Gemini, OpenAI, Anthropic,      │
                                       │ Custom cURL, Local Ollama/vLLM  │
                                       └────────────────┬────────────────┘
                                                        │
                                                        ▼
                                       ┌─────────────────────────────────┐
                                       │   Atomic Transaction Manager    │
                                       │ Snapshots /etc/config/<pkg>     │
                                       └────────────────┬────────────────┘
                                                        │
                                                        ▼
                                       ┌─────────────────────────────────┐
                                       │   AST & Capability Policy       │
                                       │ Strict token-level validation   │
                                       └────────────────┬────────────────┘
                                                        │
                                                        ▼
                                       ┌─────────────────────────────────┐
                                       │     Execution & Health Check    │
                                       │ Typed tools -> System verify    │
                                       │  -> Pass: Commit changes        │
                                       │  -> Fail: Auto-Rollback to snap │
                                       └─────────────────────────────────┘
```

---

## 📦 Installation

### Option 1: Quick Install (Recommended)

SSH into your OpenWrt router and run the universal installation one-liner:

```bash
wget -qO- --no-check-certificate https://raw.githubusercontent.com/Synapse-Works/Luci-Nexus/main/install.sh | sh
# Or using curl:
# curl -sSL https://raw.githubusercontent.com/Synapse-Works/Luci-Nexus/main/install.sh | sh
```

This installer automatically:

1. Detects your CPU architecture (`mips`, `mipsel`, `arm`, `arm64`, `amd64`).
2. Downloads and verifies the matching stripped binary into `/usr/bin/lucinexus`.
3. Configures default UCI settings in `/etc/config/lucinexus`.
4. Registers and starts the OpenWrt **Procd** service with auto-respawn protection.

### Option 2: Manual Installation via IPK

Download the package matching your router's architecture from the [Releases](https://github.com/Synapse-Works/Luci-Nexus/releases) page:

```bash
# For MIPS routers (most common Atheros/MediaTek devices):
opkg install /tmp/lucinexus_1.0.0_mips_24kc.ipk

# For ARM64 routers (Raspberry Pi 4, Banana Pi, NanoPi):
opkg install /tmp/lucinexus_1.0.0_aarch64_cortex-a53.ipk

# For x86_64 routers (mini-PCs, x86 appliances, virtual machines):
opkg install /tmp/lucinexus_1.0.0_x86_64.ipk
```

Verify installation:

```bash
lucinexus -version
# Output: LuciNexus v1.0.0 (Autonomous, Resilient OpenWrt Router Agent)
```

---

## 🏆 LuciNexus vs. LuciCodex: Architectural Comparison

| Feature / Dimension            | Legacy LuciCodex                                | **LuciNexus (Next-Gen)**                               | Why It Matters                                                           |
| ------------------------------ | ----------------------------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------ |
| **Command Execution Paradigm** | Raw shell argument arrays (`exec.Command`)      | **Strict Typed Tool Calling** (`uci_set`, `ubus_call`) | Eliminates shell metacharacter injection & command bypasses.             |
| **Security & Policy Model**    | String-joined Regex matching (`strings.Join`)   | **AST & Token Capability Matrix**                      | Argument boundaries preserved; immune to whitespace/flag bypasses.       |
| **Failure Recovery**           | Naive retry loop (repeats stderr to LLM)        | **Atomic Snapshot Rollback (`/etc/config`)**           | Never leaves router in a bricked or half-configured state.               |
| **Network Outage Tolerance**   | Fails completely if Internet is down            | **Embedded Offline Fastpath Engine**                   | Can diagnose and restore Wi-Fi/WAN without internet or API keys!         |
| **Context Memory Footprint**   | Dumps full `uci show` & logs (~60KB tokens)     | **Sparse Subsystem Context Hydration**                 | Runs smoothly on low-RAM MIPS/ARM routers (<15MB RAM).                   |
| **Custom AI Provider Setup**   | Hardcoded providers only                        | **Zero-Click cURL Import & Custom APIs**               | Paste curl snippets from DeepSeek, Groq, Ollama, OpenRouter, etc.        |
| **LLM Output Parsing**         | Brittle regex extraction on conversational text | **Native Structured JSON Schema / Tool Call API**      | 100% deterministic function calls with no conversational hallucinations. |
| **Web UI Integration**         | Basic iframe/text runner                        | **Modern Reactive LuCI JavaScript Client**             | Interactive diff preview, dry-run safety slider, instant audit.          |

## ⚙️ Configuration

LuciNexus supports **Google Gemini**, **OpenAI**, **Anthropic**, and any **Custom / Self-Hosted AI Provider**.

### 1. Zero-Click cURL Import (Any Provider)

Paste any standard `curl` command snippet from your provider's dashboard or documentation:

```bash
# DeepSeek API:
lucinexus config --curl "curl https://api.deepseek.com/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-your-key-here' \
  -d '{\"model\": \"deepseek-chat\"}'"

# Local Ollama on LAN (e.g. 192.168.1.50):
lucinexus config --curl "curl http://192.168.1.50:11434/v1/chat/completions \
  -d '{\"model\": \"llama3.2:3b\"}'"

# Groq High-Speed Inference:
lucinexus config --curl "curl https://api.groq.com/openai/v1/chat/completions \
  -H 'Authorization: Bearer gsk_your-key' \
  -d '{\"model\": \"llama-3.3-70b-versatile\"}'"
```

### 2. Standard Cloud Providers via UCI

```bash
# Google Gemini (Default):
uci set lucinexus.@api[0].provider='gemini'
uci set lucinexus.@api[0].key='YOUR_GEMINI_API_KEY'
uci set lucinexus.@api[0].model='gemini-2.5-flash'
uci commit lucinexus

# OpenAI:
uci set lucinexus.@api[0].provider='openai'
uci set lucinexus.@api[0].key='YOUR_OPENAI_API_KEY'
uci set lucinexus.@api[0].model='gpt-4o-mini'
uci commit lucinexus

# Anthropic Claude:
uci set lucinexus.@api[0].provider='anthropic'
uci set lucinexus.@api[0].key='YOUR_ANTHROPIC_API_KEY'
uci set lucinexus.@api[0].model='claude-3-5-haiku-latest'
uci commit lucinexus
```

### 3. Interactive Setup Wizard

Run the terminal setup wizard:

```bash
lucinexus setup
```

---

## 💻 Usage

### Command-Line Interface (CLI)

```bash
# 1. Safe Dry-Run Plan Preview (Default mode - no changes applied)
lucinexus "turn on 5GHz wifi and set SSID to HomeOffice"

# 2. Execute with Atomic Safety Verification & Auto-Rollback
lucinexus -approve "open port 8443 for local server 192.168.1.50"

# 3. Offline Fastpath (Zero latency, works even if WAN/Internet is down!)
lucinexus "restart wifi"
lucinexus "renew wan"
lucinexus "ping test"
lucinexus "status"

# 4. Start Background Daemon for REST API & LuCI Integration
lucinexus -server
```

### Available CLI Flags

| Flag       | Default | Description                                                      |
| :--------- | :------ | :--------------------------------------------------------------- |
| `-dry-run` | `true`  | Preview execution plan without mutating router configuration     |
| `-approve` | `false` | Execute commands with health checks and auto-rollback protection |
| `-curl`    | `""`    | Import custom AI provider configuration from a `curl` snippet    |
| `-server`  | `false` | Run in background daemon / RPC server mode (port 9999)           |
| `-config`  | `""`    | Path to custom JSON configuration file                           |
| `-version` | `false` | Display version and build information                            |

---

## 🛡️ Safety & Resilience Architecture

### 1. Atomic Transaction Snapshots (`internal/transaction`)

Multi-step configuration changes pose significant risk if interrupted. LuciNexus protects your router:

- **Point-in-Time Snapshot:** Copies affected `/etc/config` packages to `/tmp/lucinexus/snapshots/<tx_id>` before any changes are staged.
- **Non-Destructive Verification:** Executes staged tools and runs post-apply checks (gateway reachability, DNS loopback, UCI syntax parsing).
- **Automatic Rollback:** If any check fails, changes are instantly reverted (`uci revert` and snapshot restore) and services are reloaded—guaranteeing your router never bricks.

### 2. AST-Level Capability Security (`internal/policy`)

- Validates operations against strict capability roles (`read`, `network_modify`, `firewall_write`, `system_admin`).
- Enforces strict regex argument boundaries (`^[a-zA-Z0-9_.-]+$`), preventing shell metacharacter injections (`|`, `;`, `&`, `` ` ``, `$`).

### 3. Embedded Hardware Optimization

- **Sparse Fact Hydration:** Analyzes query intent to only load the required subsystem configuration (e.g., querying Wi-Fi only inspects `wireless` and `iwinfo`, omitting firewall and DHCP logs).
- **Low Memory Footprint:** Keeps memory allocation under **15MB RAM** and prompt context under **1.5KB tokens**, eliminating Out-Of-Memory (OOM) risks on 64MB/128MB devices.

---

## 🌐 LuCI Web Interface

LuciNexus includes a native OpenWrt LuCI web app (`luci-app-lucinexus`):

1. **Access:** Open your router's browser interface (e.g., `http://192.168.1.1`) and navigate to **Services → LuciNexus**.
2. **Interactive Console:** Enter plain-English requests, view planned operations in real-time, inspect configuration diffs, and approve execution with one click.
3. **cURL Import Card:** Paste cURL snippets directly into the web UI for instant connection testing and endpoint configuration.

---

## 🔨 Cross-Compilation

LuciNexus is written in Go 1.22 with zero CGO dependencies, making it simple to build for any router hardware architecture:

```bash
# Clone the repository
git clone https://github.com/Synapse-Works/Luci-Nexus.git
cd lucinexus

# Build for your specific architecture:
make build-mips      # Atheros / MediaTek MIPS (soft-float)
make build-mipsel    # MIPS little-endian
make build-arm       # ARMv7 (Cortex-A7, Cortex-A9)
make build-arm64     # ARM64 / aarch64 (Raspberry Pi 4, Cortex-A53)
make build-amd64     # x86_64 appliances & VMs

# Build all architectures simultaneously:
make build-all
```

---

## 📂 Project Structure

```text
lucinexus/
├── cmd/lucinexus/main.go            # CLI entry point, daemon, and --curl handler
├── internal/
│   ├── config/                      # UCI, environment, and cURL parser logic
│   ├── engine/                      # Execution planning, transactions, and auto-rollback
│   ├── fastpath/                    # Offline local deterministic recovery engine
│   ├── llm/                         # Structured tool-calling client (Gemini, OpenAI, Custom)
│   ├── openwrt/                     # Sparse context hydration (<15MB RAM footprint)
│   ├── policy/                      # AST & token capability matrix security
│   ├── server/                      # High-performance local REST & JSON-RPC daemon
│   ├── tools/                       # Typed OpenWrt tools (uci, service, diagnose, status)
│   └── transaction/                 # Point-in-time configuration snapshot & health checks
├── package/luci-app-lucinexus/      # LuCI Lua controller and reactive JavaScript view
├── files/                           # Procd init service and default UCI configuration
├── docs/                            # Architecture specs and deployment documentation
├── Makefile                         # Cross-compilation targets
└── install.sh                       # Universal one-line installer
```

---

## Credits

This repository basic Idea is taken from [LuciCodex](https://github.com/aezizhu/LuciCodex) by [Aezizhu](https://github.com/aezizhu)

---

## 📄 License

LuciNexus is open-source software licensed under the [Apache License, Version 2.0](https://www.google.com/search?q=LICENSE). Free for personal, homelab, and commercial deployments.
