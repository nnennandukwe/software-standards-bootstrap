# Engineering category taxonomy

Assign exactly one category to every accepted artifact. The category explains
the engineering concern, not the file type, evidence source, or tool involved.

| Category | Use when the primary concern is |
|---|---|
| `architecture` | Component boundaries, dependency direction, layering, or system structure |
| `compatibility` | Supported versions, platforms, protocols, public interfaces, or backward compatibility |
| `compliance` | License, legal, regulatory, or mandatory organizational obligations |
| `correctness` | Producing the intended result or preserving valid state and behavior |
| `developer-experience` | Making the repository easier and safer for contributors to use, build, or change |
| `documentation` | Keeping written guidance, examples, or API documentation accurate and complete |
| `maintainability` | Keeping code understandable, consistent, and safe to modify over time |
| `operability` | Deployment, configuration, observation, diagnosis, or production operation |
| `performance` | Latency, throughput, memory, storage, or computational efficiency |
| `quality` | A broad engineering-quality concern for which no narrower category is dominant |
| `reliability` | Availability, resilience, fault tolerance, recovery, or repeatable service behavior |
| `security` | Confidentiality, integrity, authentication, authorization, or vulnerability prevention |
| `testability` | Making behavior practical to verify through controlled, observable tests |

## Selection rules

1. Choose the category that best explains the artifact's engineering concern.
2. Prefer the narrowest accurate category. Use `quality` only when no narrower category fits.
3. Do not assign multiple categories.
4. Keep category independent from:
   - `lenses`, which select language, framework, task, or base context;
   - `directive`, which records behavioral modality;
   - artifact kind, which records whether the output is a rule, recipe, skill,
     or automation proposal;
   - `derivation`, which records extracted or inferred evidence;
   - `confidence`, which records evidence certainty; and
   - `scopes`, which identify affected paths.

For an Agent Skill, classify the workflow's engineering outcome rather than
the mechanics it uses. A recovery procedure may be `reliability` or
`operability`; it is not automatically `developer-experience` because a
developer runs it.
