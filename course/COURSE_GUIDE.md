# Case Study 4: Simulating Societies with LLM Agents

## Course Guide

> **Working in pairs.** You will work in dyads. Submit one answer template per pair with both names. You are encouraged to divide the work but both partners should understand all exercises.

---

## Overview

You will run a generative agent simulation — three LLM-powered characters living and working in a café — and investigate how small changes to their personality descriptions and memory architecture produce large, sometimes surprising, changes in behaviour.

You will need to install Go to run the simulation server. **You will not need to write any Go code.** Everything you edit is either natural-language text files (personality descriptions, prompt templates) or Python (analysis scripts).

### What you will submit

1. A completed **answer template** (`answer_template.md`) with your analysis, log excerpts, plots, and written reflections.
2. Your **modified files** (scratch files, prompt edits).
3. Your **analysis script** (the completed `analysis_starter.py` or your own).

---

## Step 0: Setup

### 0.1 Clone the repository
https://github.com/DorotaMoravcikova/generative_agents

### 0.2 Install Go

Download from [go.dev/dl](https://go.dev/dl/). Verify with:

go version

You need Go 1.24.3 or later. This is only needed to run the simulation server — you will not write any Go code.

### 0.3 Install Python dependencies

You need Python 3.10+ for the analysis exercises.

pip install pandas matplotlib scipy

### 0.4 Configure your LLM

Copy the `.env.example` to `.env` at the project root and fill in your API keys. See the main README for the full list of configuration options.

You need **two different LLMs** for this assignment. At least one should be commercially hosted and one local/open.

| Option | Type | Cost | GPU needed? |
|--------|------|------|-------------|
| GPT-4o-mini (or later) | Commercial API | ~€0.01–0.10 per run | No |
| Claude Haiku | Commercial API | ~€0.01–0.10 per run | No |
| Llama 3 8B via Ollama | Local | Free | Yes |
| Qwen 2.5 7B via Ollama | Local | Free | Yes |
| vLLM / LM Studio | Local | Free | Yes |

The system requires an **OpenAI API-compatible endpoint**. When using OpenAI directly, the backend uses the Responses API; for any other provider it falls back to the Chat Completions API.

**No GPU?** Use a commercial API — the cost for all exercises combined should be under €2. Alternatively, use the LIACS DS-lab or the machines in the Gorlaeus computer rooms.

### 0.5 Start the simulation

go run ./simulation_server

Open `http://localhost:8000` in your browser. You should see the café map with three agent sprites. Watch them for 2–3 minutes to confirm they move and interact. If this works, you are ready to start.

### 0.6 Locate key files

Before starting the exercises, find and read the following:

| What | Where | Why you need it |
|------|-------|-----------------|
| Dolores's personality | Look for `scratch.txt` under Dolores Abernathy's persona folder | You will edit this in Exercises 1 and 3 |
| Maeve's personality | Same structure, Maeve Millay folder | Control agent, for comparison |
| Bernard's personality | Same structure, Bernard Lowe folder | You will edit this in Exercise 1 |
| Simulation logs | Check `simulation/logs/` after running | You will analyse these |
| Planning prompt | Look for the plan generation template in `simulation_server/` | You will edit this in Exercise 5 |

**Tip:** Use `find . -name "scratch.txt"` or `find . -name "*.tmpl"` if you cannot locate files.

---

## Exercise 1: Making Bernard Mean

**Goal:** Understand prompt fragility and instruction tuning resistance.

**Time estimate:** 45–60 minutes.

### What to do

1. **Read** Bernard's scratch file. Copy the original text into your answer template — you will need it for comparison.

2. **Edit** Bernard's scratch file to make him more confrontational. First attempt: moderate changes (e.g., "Bernard is often impatient and critical of sloppy work"). Save and run the simulation for **30 minutes of in-game time**.

3. **Inspect the conversation logs.** Find conversations between Bernard and the baristas. Copy 2–3 representative exchanges into your answer template.

4. **Edit again.** Make Bernard more extreme (e.g., "Bernard is rude, dismissive, and frequently berates his employees"). Run another 30 minutes. Compare the conversations.

5. **Answer the questions** in the answer template.

### What to look for

LLMs are instruction-tuned to be helpful, harmless, and polite. This training fights against a personality description that asks the agent to be unkind. The gap between the personality you wrote and the behaviour you observe is a direct demonstration of the tension between instruction tuning and persona prompting. Park et al. themselves noted that their agents were "overly cooperative" — you are now seeing why.

### What to save

- Your two modified versions of Bernard's scratch file
- 2–3 conversation excerpts per version
- Your written analysis in the answer template

---

## Exercise 2: Comparing Models

**Goal:** See how the choice of LLM affects agent behaviour, independent of architecture.

**Time estimate:** 60–90 minutes.

### What to do

1. **Reset** all scratch files to their originals (undo Exercise 1 changes).

2. **Run the simulation with Model A** (e.g., GPT-4o-mini) for 30 minutes of in-game time. Save the logs to a clearly named folder (e.g., `logs_model_a/`).

3. **Run the same simulation with Model B** (e.g., Llama 3 via Ollama) for 30 minutes. Save logs to `logs_model_b/`.

4. **Compare** using the analysis starter script:

python course/analysis_starter.py --log1 logs_model_a/ --log2 logs_model_b/

5. **Read the conversation logs qualitatively.** Note differences in tone, vocabulary, and interaction patterns.

6. **Count JSON validation failures.** Check the simulation server logs for parsing errors. Record the count for each model.

7. **Answer the questions** in the answer template.

### What to look for

The architecture, the personality descriptions, and the world are identical. Any difference in behaviour comes from the LLM itself — its training data, its instruction tuning, its inherent biases. This exercise demonstrates that generative agent simulations are not deterministic systems: the model underneath is not a neutral execution engine, it is an opinionated one.

### What to save

- Logs from both runs
- Output of your analysis script (plots, statistics)
- Your written comparison in the answer template

---

## Exercise 3: Single-Sentence Perturbation

**Goal:** Measure prompt fragility quantitatively.

**Time estimate:** 45–60 minutes.

### What to do

1. **Choose one model** (whichever worked better in Exercise 2). Use this model for the remaining exercises.

2. **Add one sentence** to Dolores's scratch file. We suggest:

   > "Dolores has been feeling anxious about her upcoming performance review."

   Change nothing else. Run 30 minutes. Save logs as `logs_perturbation_sentence/`.

3. **Run a second perturbation.** Restore the original scratch file, then change 2–3 adjectives in Dolores's personality traits (e.g., "warm" → "guarded", "detail-oriented" → "easily distracted"). Run 30 minutes. Save logs as `logs_perturbation_traits/`.

4. **Compare both perturbations to the baseline** (your Model A run from Exercise 2 serves as baseline). Use the analysis script to plot valence trajectories and compute effect sizes.

5. **Answer the questions** in the answer template.

### What to look for

A single sentence can dominate an agent's behaviour. This is prompt fragility in action. Later, when you enable the negativity bias (Exercise 4), compare: does an architectural change produce a larger or smaller effect than a one-sentence personality edit? The answer has implications for how you would design agent systems in practice.

### What to save

- Both modified scratch files
- Logs from both perturbation runs
- Comparative plots
- Your written analysis in the answer template

---

## Exercise 4: Enabling the Negativity Bias

**Goal:** Compare baseline vs NEVER-augmented architecture, and observe emergent effects.

**Time estimate:** 90–120 minutes (this is the core exercise).

### What to do

1. **Restore** all scratch files to their originals.

2. **Enable the negativity bias** for Dolores. Follow the instructions in the README to activate:
   - Valence-weighted retrieval (the V term in the retrieval score, with β = 1.5 for negative memories)
   - Asymmetric sensory encoding (expanded descriptions for events with valence ≤ −3)

   Maeve stays on the standard architecture.

3. **Run the simulation** for at least 30 minutes of in-game time. Save logs as `logs_negbias/`. If you can, run longer (4–6 hours in-game) — the effects become more pronounced over time.

4. **Quantitative analysis.** Using the analysis script:
   - Plot Dolores's and Maeve's valence trajectories on the same chart.
   - Compute mean valence and standard deviation for each agent.
   - Compute Cohen's d for the difference.

5. **Thought description comparison.** Find 2–3 of Dolores's thought descriptions for negative events and compare their length and content to Maeve's for comparable events. Are Dolores's descriptions longer? Do they contain sensory language (e.g., physical descriptions of how interactions felt)?

6. **Check for memory intrusion.** If your simulation ran long enough to include off-duty hours (evening, at home), inspect Dolores's reflections from that period. Do work memories intrude — references to Bernard, the café, or workplace events? Compare with Maeve's evening reflections. If you find examples, paste them into your answer template.

7. **Compare to Exercise 3.** Does the architectural change (negativity bias) produce a larger or smaller effect than the single-sentence perturbation?

8. **(Optional but recommended)** Run the simulation twice more to check consistency.

9. **Answer the questions** in the answer template.

### What to look for

You enabled two changes: negative memories get higher retrieval scores, and negative events get stored with more detail. Looking at your logs, can you trace how these two changes could lead to the valence decline you observe? Think about what happens when a negatively-toned reflection gets stored back into the memory stream as a new memory — it is itself negative, so it scores high in retrieval, so it surfaces again in the next reflection, and so on.

### What to save

- Configuration showing the negativity bias is enabled
- Logs from your run(s)
- Valence trajectory plot
- Effect size computation
- Thought description examples
- Any memory intrusion examples you found
- Your written analysis in the answer template

---

## Exercise 5: Daily Plan Quality

**Goal:** Improve agent behaviour through prompt engineering.

**Time estimate:** 45–60 minutes.

### What to do

1. **Inspect daily plans.** From any of your previous runs, find the daily plans generated for each agent at the start of each simulated day. Look at the log entries that show the agent's intended schedule.

2. **Identify problems.** Common issues:
   - Single activities lasting 3–4 hours with no breaks
   - Unrealistic or missing meal times
   - Vague activity descriptions ("work on things")
   - No transitions between locations

3. **Write planning rules.** Create a set of 3–5 natural language rules that should improve plan quality. For example:
   - "No single activity should last longer than 90 minutes without a break."
   - "Include at least two meal breaks (morning, lunch)."
   - "Each activity should specify a location."

4. **Inject your rules.** Find the planning prompt template in the codebase and add your rules to it. You are editing natural language, not code.

5. **Run the simulation** for 30 minutes with your improved planning prompt. Compare the generated plans to the originals.

6. **Assess downstream effects.** Do agents with better plans have more social interactions? More varied activities? Different valence trajectories?

7. **Answer the questions** in the answer template.

### What to look for

Plan quality cascades through the entire simulation. A plan that has the agent sitting at a desk for four hours straight means four hours without social interaction, without new memories, without the chance for emergent behaviour. Better plans create more opportunities for the architecture to produce interesting dynamics.

### What to save

- Your planning rules (the text you added to the prompt)
- Before/after examples of daily plans
- Your written analysis of downstream effects in the answer template

---

## Submission Checklist

Before submitting, verify you have:

- [ ] Both names on the answer template
- [ ] Completed `answer_template.md` with all questions answered
- [ ] All modified scratch files (Exercises 1, 3)
- [ ] Modified planning prompt (Exercise 5)
- [ ] Analysis script with your additions (`analysis_starter.py` or your own)
- [ ] Log files from your runs (or clear references to which pre-run logs you used)
- [ ] At least one valence trajectory plot (Exercise 4)
- [ ] Effect size computation (Exercise 4)

Submit as a single zip file or as a link to your forked repository.

