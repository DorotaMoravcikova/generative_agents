# Good Agent Gone Bad

**When "Bad is Stronger Than Good" in the Memory of Generative Agents**

<p align="center">
  <img src="our_cover.png" alt="Good Agent Gone Bad" width="80%">
</p>

This repository accompanies *"Good Agent Gone Bad: When 'Bad is Stronger Than Good' in the Memory of Generative Agents"* (*van der Veen, *Moravčíková & van Duijn, 2026).

We implement valence-driven memory asymmetries inspired by the NEVER model (Negative Emotional Valence Enhances Recapitulation; Bowen, Kark & Kensinger, 2018) within a generative agent architecture. The original Generative Agents framework (Park et al., 2023) retrieves memories via recency, relevance, and importance — but has no mechanism for emotional valence, meaning deeply negative and deeply positive events of equal importance are treated identically. We address this gap.

The simulation runs a 48-hour workplace scenario in a three-agent café environment: an experimental agent with NEVER-inspired memory (Dolores Abernathy), a matched control (Maeve Millay), and a manager acting as a naturalistic stressor (Bernard Lowe).

## What this fork changes

This codebase is a substantially modified fork of [Park et al.'s Generative Agents](https://github.com/joonspk-research/generative_agents). 

Key changes:
**Valence-aware memory retrieval**

We add a valence dimension to the retrieval score. Absolute valence is normalised to [0, 1] and weighted ×1.5 for negative memories, producing a retrieval hierarchy (negative > positive > neutral). The full retrieval score is: score = α_rec·R + α_rel·L + α_imp·I + α_val·V (all α = 1).

**Asymmetric sensory encoding**

Events scored ≤ −3 on a [−10, +10] valence scale are stored with expanded perceptual detail, following the NEVER model's prediction that negative events trigger richer sensory encoding.


## Setup

<!-- TODO: fill in -->

## Citation

If you use this code, please cite both our paper and the original Generative Agents work:

```bibtex
@article{VanDerVeen2026GoodAgent,
  author  = {van der Veen, Friso B.H. and Moravčíková, Dorota and van Duijn, Marc J.},
  title   = {Good Agent Gone Bad: When `Bad is Stronger Than Good' in the Memory of Generative Agents},
  year    = {2026}
}

@inproceedings{Park2023GenerativeAgents,
  author    = {Park, Joon Sung and O'Brien, Joseph C. and Cai, Carrie J. and Morris, Meredith Ringel and Liang, Percy and Bernstein, Michael S.},
  title     = {Generative Agents: Interactive Simulacra of Human Behavior},
  year      = {2023},
  publisher = {Association for Computing Machinery},
  booktitle = {Proceedings of the 36th Annual ACM Symposium on User Interface Software and Technology (UIST '23)},
  location  = {San Francisco, CA, USA},
  series    = {UIST '23}
}
```

## Acknowledgements

This project builds on the [Generative Agents](https://github.com/joonspk-research/generative_agents) framework by Joon Sung Park, Joseph C. O'Brien, Carrie J. Cai, Meredith Ringel Morris, Percy Liang, and Michael S. Bernstein.

The original game assets:
[PixyMoon](https://twitter.com/_PixyMoon_) (backgrounds), [LimeZu](https://twitter.com/lime_px) (furniture/interiors), and [ぴぽ](https://twitter.com/pipohi) (characters).

## License

This project retains the [MIT License](LICENSE) from the original Generative Agents repository.
