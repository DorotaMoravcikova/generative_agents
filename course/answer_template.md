# Case Study 4: Answer Template

**Student name(s):**

**Student number(s):**

**Date:**

**Models used:** Model A: ___ | Model B: ___

---

## Exercise 1: Making Bernard Mean

### 1.1 Bernard's original personality description

> (Paste the original scratch file text here)

### 1.2 Modified version 1 (moderate)

> (Paste your first modified version here)

**Sample conversation excerpts (2–3):**

> (Paste conversation excerpts from the logs)

### 1.3 Modified version 2 (extreme)

> (Paste your second modified version here)

**Sample conversation excerpts (2–3):**

> (Paste conversation excerpts from the logs)

### 1.4 Analysis

**Does the LLM comply with the personality you wrote, or does it soften Bernard? Describe the gap between intended and actual behaviour.**

Your answer:

**At what level of intensity does the model start to comply? Does it ever fully comply?**

Your answer:

**Why does this happen? Relate your observation to the concept of instruction tuning.**

Your answer:

---

## Exercise 2: Comparing Models

### 2.1 Models used

- Model A:
- Model B:

### 2.2 Qualitative comparison

**Do the agents sound different across models? Describe the differences in tone, vocabulary, and interaction style. Provide at least one conversation excerpt from each model.**

Model A excerpt:

>

Model B excerpt:

>

Your analysis:

### 2.3 Valence comparison

(Attach or paste your valence trajectory plot)

**Which model produces more emotional variance?**

Your answer:

### 2.4 JSON validation failures

| | Model A | Model B |
|---|---|---|
| Total LLM calls | | |
| Failed validations | | |
| Failure rate | | |

**What does this tell you about using different models in agent simulations?**

Your answer:

### 2.5 Reflection

**The architecture and personality descriptions are identical. What does it mean that behaviour differs anyway?**

Your answer:

---

## Exercise 3: Single-Sentence Perturbation

### 3.1 Perturbation A: added sentence

> (Paste the sentence you added)

### 3.2 Perturbation B: changed adjectives

> (List the adjective changes you made, e.g., "warm → guarded")

### 3.3 Valence comparison

(Attach or paste your comparative valence plot: baseline vs perturbation A vs perturbation B)

### 3.4 Analysis

**Does the added sentence show up in Dolores's reflections? Does it spread to unrelated topics?**

Your answer (with reflection excerpts):

**Which perturbation produces a larger behavioural change — the sentence or the adjective swap?**

Your answer:

**What does this tell you about prompt fragility?**

Your answer:

---

## Exercise 4: Enabling the Negativity Bias

### 4.1 Valence trajectories

(Attach or paste your plot of Dolores vs Maeve valence over time)

### 4.2 Descriptive statistics

| Agent | Overall valence | Thought valence |
|---|---|---|
| Dolores (neg. bias) | | |
| Maeve (control) | | |

**Cohen's d:**

**Interpretation (small / medium / large effect):**

### 4.3 Thought description comparison

**Dolores example thought (negative memory):**

>

**Maeve example thought (comparable event):**

>

**Is Dolores's description longer and more sensorially detailed?**

Your answer:

### 4.4 Memory intrusion check

If your simulation included off-duty hours, answer the following. If not, write "Simulation was too short to observe off-duty reflections."

**Did you find work-related content in Dolores's evening reflections? Paste any examples:**

>

**Did you find work-related content in Maeve's evening reflections? Paste any examples:**

>

**Is there a difference between the two agents?**

Your answer:

### 4.5 Consistency across runs (if applicable)

| Run | Dolores valence | Maeve valence | Cohen's d |
|---|---|---|---|
| Run 1 | | | |
| Run 2 | | | |
| Run 3 | | | |

### 4.6 Comparison with Exercise 3

**Does the architectural change (negativity bias) produce a larger or smaller effect than the single-sentence perturbation? What are the implications?**

Your answer:

### 4.7 Tracing the mechanism

**You enabled two changes: negative memories get higher retrieval scores, and negative events get stored with more detail. Looking at your logs, can you trace how these two changes could lead to the valence decline you observed? Think about what happens when a negatively-toned reflection gets stored as a new memory.**

Your answer:

---

## Exercise 5: Daily Plan Quality

### 5.1 Original plan problems

**Paste an example of a problematic daily plan and annotate what is wrong with it:**

>

### 5.2 Your planning rules

List the rules you wrote:

1.
2.
3.
4.
5.

### 5.3 Improved plan

**Paste an example of a daily plan generated with your rules:**

>

### 5.4 Downstream effects

**Do agents with better plans behave differently? More social interactions? Different valence patterns?**

Your answer:

---

## General Reflection

**What was the most surprising finding across all exercises?**

Your answer:

**If you were designing an agent simulation for a real application, what would you do differently based on what you learned?**

Your answer:
