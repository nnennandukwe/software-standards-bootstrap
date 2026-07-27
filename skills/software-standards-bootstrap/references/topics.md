# Primary engineering topic taxonomy

Assign exactly one primary topic to every rule and generated Agent Skill. The topic should explain the main engineering risk, obligation, or outcome—not merely the file type, evidence source, or tool involved.

| Topic | Use when the primary concern is |
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
| `quality` | A broad engineering-quality concern for which no narrower topic is dominant |
| `reliability` | Availability, resilience, fault tolerance, recovery, or repeatable service behavior |
| `security` | Confidentiality, integrity, authentication, authorization, or vulnerability prevention |
| `testability` | Making behavior practical to verify through controlled, observable tests |

## Selection rules

1. Choose the topic that best explains why a developer must follow the rule or workflow.
2. Prefer the narrowest accurate topic. Use `quality` only when no narrower topic fits.
3. Do not assign multiple topics. If several concerns apply, choose the dominant one and explain the tradeoff in the assessment.
4. Keep topic independent from:
   - `lenses`, which select language, framework, task, or base context;
   - `directive`, which records behavioral modality;
   - `classification`, which records whether proof is guidance or deterministic;
   - `importance`, which comes from the score;
   - `confidence`, which records evidence certainty; and
   - `scopes`, which identify affected paths.

For an Agent Skill, classify the workflow's primary engineering outcome rather than the mechanics it uses. A release procedure may be `reliability` or `operability`; it is not automatically `developer-experience` merely because a developer runs it.
