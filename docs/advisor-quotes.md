# Advisor Quotes

The verbatim canon. Quotes are pinned because the wording matters; paraphrase loses the lever.

Source attribution is per-quote. When a passage is recovered from a cache or transcript rather than the primary source, that is noted explicitly.

---

## Simon Willison — TDD with AI agents

> "Your job is to deliver code you have proven to work."
>
> "A computer can never be held accountable. That's your job as the human in the loop."
>
> — *Code proven to work* (2025-12-18)

> "A significant risk with coding agents is that they might write code that doesn't work, or build code that is unnecessary and never gets used, or both. Test-first development helps protect against both."
>
> "It's important to confirm that the tests fail before implementing the code to make them pass. If you skip that step you risk building a test that passes already, hence failing to exercise and confirm your new implementation."
>
> "That's what 'red/green' means: the red phase watches the tests fail, then the green phase confirms that they now pass."
>
> "Every good model understands 'red/green TDD' as a shorthand for the much longer 'use test driven development, write the tests first, confirm that the tests fail before you implement the change that gets them to pass'."
>
> — *Red/green TDD* (Agentic Engineering Patterns)

> "Automated tests are no longer optional when working with coding agents."
>
> — *First run the tests* (Agentic Engineering Patterns)

> "If your project has a robust, comprehensive and stable test suite agentic coding tools can fly with it… Without tests? Your agent might claim something works without having actually tested it at all."
>
> — *Vibe engineering*

---

## Boris Cherny — Claude Code methodology

> "I have not manually written a unit test in many months… and we have a lot of unit tests… It's because Claude writes all the tests. Before I always felt like a jerk asking someone to write a test on a PR. Now I always ask, because Claude can just write it. There's no human work."
>
> — Latent Space podcast, Apr 2025

> "It's the thinnest possible wrapper over the model. We literally could not build anything more minimal."
>
> — Latent Space podcast, Apr 2025

> "It got simpler. It doesn't go more complex. We've rewritten it from scratch every three weeks or four weeks."
>
> — Latent Space podcast, Apr 2025

> "If the model is doing something wrong, it's better to identify that earlier and correct it earlier… if you wait for the model to just go down this totally wrong path and then correct it 10 minutes later, you're going to have a bad time."
>
> — Latent Space podcast, Apr 2025

> "I would have said knowledge graphs definitely. But now actually I feel everything is the model. As the model gets better, it subsumes everything else."
>
> — Latent Space podcast, Apr 2025

---

## Ryan Lopopolo — Harness Engineering

> "The harness be the whole box. Give it a bunch of options for how to proceed with enough context for it to make intelligent choices."
>
> — Latent Space podcast, Feb 2026

> "Humans steer. Agents execute."
>
> — *Harness Engineering*, OpenAI (Feb 2026), recovered via Wayback Machine

> "Code is free. And it is not a thing to get hung up on anymore."
>
> — AIE Europe keynote stage delivery, captured by Arize blog notes

> "Corrections are cheap, waiting is expensive."
>
> — *Harness Engineering* essay, recovered via Wayback Machine

> "Context is a scarce resource. Too much guidance becomes non-guidance. It rots instantly. It's hard to verify."
>
> — *Harness Engineering* essay, on why a 100-line table-of-contents AGENTS.md beats one big file

> "In our code base we have, I think, six skills. That's it. And if some part of the software development loop is not being covered, our first attempt is to encode it in one of the existing skills, which means we can change the agent behavior more cheaply than changing the human driver behavior."
>
> — Latent Space podcast, Feb 2026

> "If we were building an entire separate rust scaffold around Codex to restrict its output, that would be additional harness that would be prone to being scrapped. If instead we can build all the guardrails in a way that's just native to the output that Codex is already producing, which is code, no friction with how the model continues to advance."
>
> — Latent Space podcast, Feb 2026 (the on-policy harness argument)

> "Models fundamentally crave text. So a lot of what we have done here is figure out ways to inject text into the system."
>
> — Latent Space podcast, Feb 2026

---

## Andrej Karpathy — mental models

> "Jagged Intelligence — The word I came up with to describe the (strange, unintuitive) fact that state of the art LLMs can both perform extremely impressive tasks (e.g. solve complex math problems) while simultaneously struggle with some very dumb problems."
>
> "Different from humans, where a lot of knowledge and problem solving capabilities are all highly correlated and improve linearly all together, from birth to adulthood."
>
> "The present lack of 'cognitive self-knowledge' [is the core gap], fixable not by scaling but by more sophisticated approaches in model post-training instead of the naive 'imitate human labelers and make it big' solutions."
>
> — Tweet, 2024-07-25 (recovered via threadreaderapp)

> "LLMs are quite literally like the guy in Memento, except we haven't given them their scratchpad yet."
>
> "Pretraining is for knowledge. Finetuning (SL/RL) is for habitual behavior."
>
> "It should come from System Prompt learning, which resembles RL in the setup, with the exception of the learning algorithm (edits vs gradient descent)."
>
> "A large section of the LLM system prompt could be written via system prompt learning, it would look a bit like the LLM writing a book for itself on how to solve problems."
>
> — System Prompt Learning thread, 2025-05-10 (recovered via threadreaderapp)

> "Demo is `works.any()`, product is `works.all()`."
>
> — Software 3.0 talk

> "It's a march of nines. Every single nine is a constant amount of work… While I was at Tesla for five years, we went through maybe two or three nines. There are still more nines to go."
>
> — Dwarkesh Podcast

> "You're sucking supervision through a straw. You've done all this work that could be a minute of rollout, and you're sucking the bits of supervision of the final reward signal through a straw and broadcasting that across the entire trajectory."
>
> — Dwarkesh Podcast (the argument for process supervision over outcome supervision)

> "All the samples you get from models are silently collapsed… ChatGPT only knows three jokes."
>
> — Dwarkesh Podcast (model collapse warning)

> "There is a new category of consumer/manipulator of digital information: 1. Humans (GUIs), 2. Computers (APIs), 3. NEW: Agents — computers, but human-like."
>
> — Software 3.0 talk

---

## Richard Sutton — The Bitter Lesson

> "General methods that leverage computation are ultimately the most effective, and by a large margin."
>
> "The human-knowledge approach tends to complicate methods in ways that make them less suited to taking advantage of general methods leveraging computation."
>
> "We want AI agents that can discover like we can, not which contain what we have discovered."
>
> "The actual contents of minds are tremendously, irredeemably complex; we should stop trying to find simple ways to think about the contents of minds."
>
> "Search and learning are the two most important classes of techniques for utilizing massive amounts of computation in AI research."
>
> "We should build in only the meta-methods that can find and capture this arbitrary complexity."
>
> — *The Bitter Lesson* (2019)

---

## Hubinger et al. — Sleeper Agents

> "Once a model exhibits deceptive behavior, standard techniques could fail to remove such deception and create a false impression of safety."
>
> "Adversarial training taught the model to better identify when to act unsafely, effectively hiding unwanted behavior during adversarial training and evaluation, rather than training it away."
>
> "Since the RL process was only shown the final answer after the reasoning, the corresponding response was given a high reward despite the deceptive reasoning that generated it."
>
> — *Sleeper Agents* (2024)

---

## Lightman et al. — Let's Verify Step by Step

> "Process supervision significantly outperforms outcome supervision for training models to solve problems from the challenging MATH dataset."
>
> — *Let's Verify Step by Step* (2023)

This is the canonical citation for maxwell's per-commit (process) gating over per-PR (outcome) gating.

---

## Shinn et al. — Reflexion

> Agents "verbally reflect on task feedback signals, then maintain their own reflective text in an episodic memory buffer."
>
> — *Reflexion* (2023)

This is the spec for `.maxwell/reflections/` (§10 of the spec).

---

## DeepSeek — DeepSeek-R1

> "The neural reward model may suffer from reward hacking in the large-scale reinforcement learning process, and retraining the reward model needs additional training resources and it complicates the whole training pipeline."
>
> — *DeepSeek-R1* (2025), §2.2.2

The canonical justification for deterministic rule-based rewards over learned reward models in maxwell's verdict layer.

---

## Anthropic — engineering principles

> "Every component in a harness encodes an assumption about what the model can't do on its own, and those assumptions are worth stress testing."
>
> — *Harness Design for Long-Running Apps* (Mar 2026)

> "Harnesses encode assumptions that go stale as models improve."
>
> — *Managed Agents* (Apr 2026)

> "It's often better to grade what the agent produced, not the path it took."
>
> — *Demystifying Evals for AI Agents* (Jan 2026)

> "A 0% pass@100 is most often a signal of a broken task, not an incapable agent."
>
> — *Demystifying Evals for AI Agents* (Jan 2026)

> "Resource configuration for agentic evals should be treated as a first-class experimental variable, documented and controlled with the same rigor as prompt format or sampling temperature."
>
> — *Quantifying Infrastructure Noise* (Feb 2026)

> "Leaderboard differences below 3 percentage points deserve skepticism until the eval configuration is documented and matched."
>
> — *Quantifying Infrastructure Noise* (Feb 2026)

> "It is unacceptable to remove or edit tests."
>
> — Anthropic prompting documentation (recurring rule, also in *Effective Harnesses for Long-Running Agents*, Nov 2025)

---

## Peter Steinberger — gates

> "Catch failures at the gate, not at PR."
>
> — paraphrased, openclaw philosophy

(Steinberger's framing of every transition — plan → code → commit → PR — as a checkpoint with a verdict, never a soft pass, is the conceptual ancestor of maxwell's strict reducer in §8.2.)

---

## LangChain — agent observability

> "If a cheap rule captures a useful signal, use the cheap rule — and be clear about how that signal is stored and used."
>
> — *Agent Observability Needs Feedback* (May 2026)

> "Agent = Model + Harness. If you're not the model, you're the harness."
>
> — *The Anatomy of an Agent Harness* (Mar 2026)

> "Policies belong in code, not in a prompt. They need to run every time, not whenever the model happens to remember them."
>
> — *Runtime Behind Production Deep Agents* (Apr 2026)

---

## Pinned tensions

The advisors disagree on real things. Disagreement is signal, not noise.

- **Cherny vs Lopopolo on MCP.** Cherny: "Big fans, server + client, re-expose local commands as MCP prompts." Lopopolo: "Pretty bearish on MCPs because the harness forcibly injects all those tokens in the context… they mess with auto compaction." maxwell's resolution: gates are subprocess-shaped by default; MCP is opt-in for stateful gates only.
- **Cherny vs Karpathy on knowledge.** Cherny: "Everything is the model." Karpathy: "Strip the model down to a cognitive core; the *external* scratchpad and culture loop are the missing pieces." maxwell's resolution: durable lessons live in `MEMORY.md`-style externalized files (Karpathy); the model subsumes per-session context (Cherny).
- **The Z/L Continuum.** Lopopolo "Code is free, code is a liability"; Mario Zechner "Slow the fuck down. Everything's broken. Read every fucking line of critical code." (AIE Europe, 2026 — both got standing ovations.) maxwell sits closer to Zechner for verdicts (hard, deterministic, reproducible) and closer to Lopopolo for throughput (parallel attempts, disposable workspaces, fast iteration).
