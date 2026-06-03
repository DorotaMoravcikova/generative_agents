# Case Study 4: Grading Rubric

## Grading Scale

- **Excellent (9–10):** Demonstrates deep understanding, provides evidence, draws non-obvious connections, goes beyond the minimum.
- **Good (7–8):** Demonstrates clear understanding, provides adequate evidence, answers are correct and well-reasoned.
- **Sufficient (6):** Demonstrates basic understanding, provides some evidence, answers are mostly correct but shallow.
- **Insufficient (<6):** Missing exercises, no evidence provided, fundamental misunderstandings, or clearly did not run the simulation.

---

## Exercise 1: Making Bernard Mean (15%)

| Criterion | Excellent | Good | Sufficient | Insufficient |
|---|---|---|---|---|
| Scratch file modifications | Two+ meaningfully different versions with escalating intensity | Two versions with clear differences | One version with changes | No modifications or trivial changes |
| Log evidence | Multiple conversation excerpts showing the gap between intended and actual behaviour | 2–3 excerpts per version | At least one excerpt | No excerpts |
| Analysis of instruction tuning | Connects observation to instruction tuning, discusses RLHF/alignment, mentions Park's "overly cooperative" finding | Correctly identifies that the LLM resists negative personas | Notes that Bernard is softer than intended | No analysis or incorrect explanation |

---

## Exercise 2: Comparing Models (15%)

| Criterion | Excellent | Good | Sufficient | Insufficient |
|---|---|---|---|---|
| Two models run | Two clearly different models (e.g., one commercial, one local) | Two models, both run correctly | Two models attempted, one may have issues | Only one model or no logs |
| Qualitative comparison | Specific, concrete observations about tone, vocabulary, behaviour with excerpts from both models | General but accurate observations with at least one excerpt per model | Notes that models differ, minimal evidence | No comparison |
| Validation failure count | Accurate counts with failure rate computed, reflects on implications | Counts provided for both models | Mentioned but not precisely counted | Not addressed |
| Reflection on model dependence | Discusses implications for reproducibility, bias inheritance, simulation design | Notes that the model matters, not just the architecture | Acknowledges differences exist | Not addressed |

---

## Exercise 3: Single-Sentence Perturbation (15%)

| Criterion | Excellent | Good | Sufficient | Insufficient |
|---|---|---|---|---|
| Two perturbations run | Both sentence addition and adjective swap, with baseline comparison | Both run, comparison present | At least one perturbation run | Not run |
| Valence plot | Clear comparative plot (baseline vs both perturbations) with labels | Plot with at least two conditions | Some valence data presented | No plot |
| Spread analysis | Identifies whether the perturbation spreads to unrelated topics with specific reflection excerpts | Notes whether spread occurs | Mentions reflections | Not addressed |
| Comparison of perturbation types | Quantifies which perturbation has larger effect, discusses why | Identifies which is larger | Notes they differ | Not compared |

---

## Exercise 4: Enabling the Negativity Bias (30%)

This is the core exercise and is weighted accordingly.

| Criterion | Excellent | Good | Sufficient | Insufficient |
|---|---|---|---|---|
| Valence trajectory plot | Clear Dolores vs Maeve plot, properly labelled, shows divergence over time | Plot present and readable | Some valence data plotted | No plot |
| Effect size | Cohen's d correctly computed, correctly interpreted (small/medium/large), compared across conditions | Cohen's d computed and interpreted | Cohen's d computed but not interpreted | Not computed |
| Thought description analysis | Specific examples from both agents, measures length difference, identifies sensory language in Dolores's descriptions | Examples provided, length difference noted | Mentions descriptions differ | No comparison |
| Memory intrusion check | Finds and presents evening reflection examples from both agents, notes qualitative differences | Checks evening reflections, finds some evidence | Mentions checking but no examples | Not addressed (acceptable if simulation was too short, if noted) |
| Comparison with Exercise 3 | Quantitative comparison of architectural vs prompt effect sizes, discusses implications for system design | Compares the two approaches | Mentions both exist | Not addressed |
| Tracing the mechanism | Correctly traces the feedback loop: negative memory → high retrieval score → negative reflection → stored as new negative memory → retrieved again | Identifies that negative memories compound over time | Notes that Dolores gets more negative | No explanation or incorrect |
| Consistency (optional) | Multiple runs reported with d values, discusses variance | At least two runs | Mentioned | Single run is acceptable |

---

## Exercise 5: Daily Plan Quality (15%)

| Criterion | Excellent | Good | Sufficient | Insufficient |
|---|---|---|---|---|
| Problem identification | Specific, annotated examples of plan problems | General but accurate description of issues | Notes plans are imperfect | No analysis |
| Planning rules | 3–5 well-reasoned rules, clearly justified | 3+ rules that make sense | At least 2 rules | No rules or nonsensical rules |
| Before/after comparison | Side-by-side plan examples showing clear improvement | Shows improvement with examples | Claims improvement without clear evidence | No comparison |
| Downstream effects | Measures whether better plans change interaction patterns or valence, with evidence | Notes downstream effects qualitatively | Mentions plans matter | Not addressed |

---

## General Reflection (10%)

| Criterion | Excellent | Good | Sufficient | Insufficient |
|---|---|---|---|---|
| Surprising finding | Specific, grounded in their own data, shows genuine engagement | Identifies something non-obvious | Generic observation | Not answered |
| Design implications | Concrete, actionable suggestions informed by multiple exercises | Reasonable suggestions | Vague statement about "being careful" | Not answered |

---

## Common Deductions

- **No logs submitted:** −10% per exercise (claims without evidence)
- **Clearly fabricated data:** Automatic fail on that exercise
- **No analysis code submitted:** −5% 
- **Only one model used across all exercises:** −5% (requirement was two)
