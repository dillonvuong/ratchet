# References

This is the living index of sources that informed the spec. Each entry maps to one or more spec sections so a reviewer can trace any normative claim to its origin.

URLs rot. When a primary source goes offline, prefer the Wayback Machine snapshot and record both URLs.

## Tier 0 — Foundational for the project

- **Sutton 2019, *The Bitter Lesson*** — basis for §13.9.
  https://incompleteideas.net/IncIdeas/BitterLesson.html
- **Karpathy LLM Wiki gist** — structural inspiration for this whole project.
  https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f

## Tier 1 — TDD canon

- **Willison, *Red/green TDD*** — basis for the TDD gate canon.
  https://simonwillison.net/guides/agentic-engineering-patterns/red-green-tdd/
- **Willison, *First run the tests*** — basis for the doctrine prompt's session-opener.
  https://simonwillison.net/guides/agentic-engineering-patterns/first-run-the-tests/
- **Willison, *Code proven to work*** — basis for the verdict-language framing.
  https://simonwillison.net/2025/Dec/18/code-proven-to-work/
- **Willison, *Vibe engineering*** — adjacent canon.
  https://simonw.substack.com/p/vibe-engineering

## Tier 2 — Agent research papers (gate model)

- **Lightman et al. 2023, *Let's Verify Step by Step*** — basis for per-gate process supervision.
  https://arxiv.org/abs/2305.20050
- **Shinn et al. 2023, *Reflexion*** — basis for §10 (reflection artifact).
  https://arxiv.org/abs/2303.11366
- **Yao et al. 2022, *ReAct*** — basis for thought-action-observation traces in transcripts.
  https://arxiv.org/abs/2210.03629
- **Schick et al. 2023, *Toolformer*** — basis for "downstream-validated tool call" pattern; future evaluator gate.
  https://arxiv.org/abs/2302.04761
- **Wang et al. 2024, *CodeAct*** — basis for subprocess-shaped (vs JSON-tool-shaped) gate API.
  https://arxiv.org/abs/2402.01030
- **DeepSeek 2025, *DeepSeek-R1*** — basis for deterministic verifiable rewards.
  https://arxiv.org/abs/2501.12948
- **Hubinger et al. 2024, *Sleeper Agents*** — basis for §13.1 and §13.2.
  https://arxiv.org/abs/2401.05566

## Tier 3 — Harness specs

- **Symphony SPEC.md** — structural model for this specification.
  https://github.com/openai/symphony/blob/main/SPEC.md
- **Lopopolo, *Harness Engineering*** (OpenAI, Feb 2026) — basis for AGENTS.md table-of-contents rule and on-policy harness argument.
  https://openai.com/index/harness-engineering/
  Wayback: https://web.archive.org/web/20260211221555/https://openai.com/index/harness-engineering/
- **Anthropic, *Harness Design for Long-Running Apps*** (Mar 2026) — basis for stress-test rule and generator-evaluator pattern.
  https://www.anthropic.com/engineering/harness-design-long-running-apps
- **Anthropic, *Effective Harnesses for Long-Running Agents*** (Nov 2025) — basis for `feature_list.json` analog and session-bearings pattern.
  https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents
- **Anthropic, *Effective Context Engineering for AI Agents*** (Sep 2025) — basis for context-rot framing and tool-design rule.
  https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents
- **Anthropic, *Building Effective Agents*** (Dec 2024) — basis for poka-yoke tool design and "do simple thing" framing.
  https://www.anthropic.com/research/building-effective-agents
- **Anthropic, *Managed Agents*** (Apr 2026) — basis for Brain/Hands/Session reserved architecture in §3.2.
  https://www.anthropic.com/engineering/managed-agents

## Tier 4 — Eval methodology

- **Anthropic, *Demystifying Evals for AI Agents*** (Jan 2026) — basis for Pass@k/Pass^k in §8.4 and capability/regression suite distinction.
  https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents
- **Anthropic, *Quantifying Infrastructure Noise*** (Feb 2026) — basis for §14.4 (3× hard-limit rule, 3pp leaderboard skepticism).
  https://www.anthropic.com/engineering/infrastructure-noise
- **Anthropic, *AI-Resistant Technical Evaluations*** (Jan 2026) — basis for using Claude itself as a difficulty oracle when designing future task corpora.
  https://www.anthropic.com/engineering/AI-resistant-technical-evaluations
- **Anthropic, *Building a C Compiler with Parallel Claudes*** (Feb 2026) — basis for `current_tasks/<lock>.txt` git-locking pattern (reserved for v1+).
  https://www.anthropic.com/engineering/building-c-compiler
- **SWE-Bench** — basis for FULL/PARTIAL/NO ladder and F2P/P2P split.
  https://www.swebench.com/
- **Terminal-Bench 2.0** — basis for `reward.txt`/`reward.json` task-runner output contract (reserved).
  https://www.tbench.ai/
- **LiveCodeBench** — basis for flake-budget rationale (<0.5pt variance).
  https://livecodebench.github.io/

## Tier 5 — Open standards (portability foundation)

- **AGENTS.md** — basis for §5.3.
  https://agents.md
- **Codex AGENTS.md guide** — host-specific extension behavior.
  https://developers.openai.com/codex/guides/agents-md
- **SKILL.md spec** — basis for §5.4.
  https://agentskills.io/specification
- **Anthropic, *Equipping Agents for the Real World with Agent Skills*** (Oct 2025) — design rationale for skills.
  https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills
- **MCP Spec 2025-11-25** — reserved for opt-in stateful gate transport.
  https://modelcontextprotocol.io/specification/2025-11-25
- **Agent2Agent (A2A) Protocol** — reserved adapter namespace; `Task` lifecycle informs verdict semantics.
  https://a2a-protocol.org/latest/specification/
- **Agent Client Protocol (ACP)** — reserved adapter namespace; editor-seam pattern.
  https://agentclientprotocol.com/
- **Cursor Rules** — host-specific shim source.
  https://docs.cursor.com/context/rules

## Tier 6 — Methodology / advisor canon

- **Latent Space — Boris Cherny on Claude Code** (Apr 2025) — Cherny methodology canon.
  https://www.latent.space/p/claude-code
- **Latent Space — Lopopolo: Extreme Harness Engineering** (Feb 2026) — Lopopolo methodology canon.
  https://www.latent.space/p/harness-eng
- **Latent Space — Karpathy "Software 3.0"** (Jun 2025) — third-consumer framing.
  https://www.latent.space/p/s3
- **Dwarkesh Podcast — Karpathy** — march of nines, model collapse, RL critique.
  https://www.dwarkesh.com/p/andrej-karpathy
- **Karpathy, jagged intelligence thread** (2024-07-25). Recovered via threadreaderapp.
  https://x.com/karpathy/status/1816531576228053133
- **Karpathy, system prompt learning thread** (2025-05-10). Recovered via threadreaderapp.
  https://x.com/karpathy/status/1921368644069765486
- **Karpathy, anterograde amnesia reply** to Dwarkesh.
  https://x.com/karpathy/status/1930003172246073412
- **LangChain, *The Anatomy of an Agent Harness*** (Mar 2026) — middleware-hooks framing.
  https://www.langchain.com/blog/the-anatomy-of-an-agent-harness
- **LangChain, *Runtime Behind Production Deep Agents*** (Apr 2026) — `before_model`/`wrap_tool_call` hooks.
  https://www.langchain.com/blog/runtime-behind-production-deep-agents
- **LangChain, *Agent Observability Needs Feedback*** (May 2026) — "if a cheap rule captures useful signal, use the cheap rule."
  https://www.langchain.com/blog/agent-observability-needs-feedback-to-power-learning

## Tier 7 — Safety / alignment

- **Anthropic, *Core Views on AI Safety*** — basis for §13.7 and §13.8.
  https://www.anthropic.com/news/core-views-on-ai-safety
- **Irving, Christiano, Amodei 2018, *AI Safety via Debate*** — basis for reserved `gates.debate.*`.
  https://arxiv.org/abs/1805.00899
- **METR** — basis for `human_baseline_minutes` and run-artifact requirements.
  https://metr.org/
- **METR, *Measuring AI Ability to Complete Long Tasks***.
  https://metr.org/blog/2025-03-19-measuring-ai-ability-to-complete-long-tasks/
- **UK AI Security Institute** — basis for safety-case framing.
  https://www.aisi.gov.uk/work
- **Mathematical Framework for Transformer Circuits** (Anthropic 2021) — reserved `gates.interp.*`.
  https://transformer-circuits.pub/2021/framework/index.html

## Tier 8 — Foundational ML (for completeness; orthogonal to gate design)

- **Vaswani et al. 2017, *Attention Is All You Need*** — ablation methodology informs how we add gates.
  https://arxiv.org/abs/1706.03762
- **Brown et al. 2020, *GPT-3*** — contamination measurement methodology.
  https://arxiv.org/abs/2005.14165
- **Hoffmann et al. 2022, *Chinchilla*** — equal-scaling argument for gate complexity vs task corpus.
  https://arxiv.org/abs/2203.15556
- **Ouyang et al. 2022, *InstructGPT*** — pairwise preference is for taste-domains only.
  https://arxiv.org/abs/2203.02155
- **Bai et al. 2022, *Constitutional AI*** — critique-revise mechanism informs refactor phase.
  https://arxiv.org/abs/2212.08073
- **Rafailov et al. 2023, *DPO*** — "the policy is the reward model" → "the gate suite is the spec".
  https://arxiv.org/abs/2305.18290

## Tier 9 — Reference implementations to clone (read-only)

- **openai/symphony** — Apache 2.0 reference impl.
  https://github.com/openai/symphony
- **openai/codex** — codex-rs/app-server is the canonical hook integration reference.
  https://github.com/openai/codex
- **anthropics/skills** — skill-creator is the canonical SKILL.md reference.
  https://github.com/anthropics/skills
- **anthropics/claude-agent-sdk-python** — pattern source for adapter; we do not depend on it.
  https://github.com/anthropics/claude-agent-sdk-python
- **anthropics/claude-quickstarts** — autonomous-coding/client.py is the canonical `.claude_settings.json` reference.
  https://github.com/anthropics/claude-quickstarts
- **modelcontextprotocol/servers** — reference implementations only; do not vendor.
  https://github.com/modelcontextprotocol/servers
- **langchain-ai/deepagents** — middleware patterns to mirror.
  https://github.com/langchain-ai/deepagents
- **UKGovernmentBEIS/inspect_ai** — `Scorer` abstractions we mirror in Go.
  https://github.com/UKGovernmentBEIS/inspect_ai

## Recovery notes

The following primary URLs returned 403 / SSL errors / Cloudflare blocks during initial research and were recovered via Wayback Machine snapshots:

- `openai.com/index/harness-engineering/` → Wayback `20260211221555`.
- `openai.com/index/unlocking-the-codex-harness/` → Wayback `20260214083730`.
- `openai.com/index/open-source-codex-orchestration-symphony/` → Wayback `20260428094656`.
- `openai.com/index/learning-to-reason-with-llms/` → Wayback `20240912171926`.
- `openai.com/index/new-tools-for-building-agents/` → Wayback `20250311221124`.
- `x.com/karpathy/status/<id>` → threadreaderapp.com unrolls.

The AIE Europe Lopopolo keynote video (`youtube.com/watch?v=CeOXx-XTYek`) was inaccessible to the research environment; substantive content was recovered from the Latent Space transcript, ThursdAI eyewitness recap, and Arize blog stage notes.

## CI

`.github/workflows/check-references.yml` runs a link checker on every PR. Broken links are flagged but do not fail the build. A weekly job opens an issue listing currently-broken links.
