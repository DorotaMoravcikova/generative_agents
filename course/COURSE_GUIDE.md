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
