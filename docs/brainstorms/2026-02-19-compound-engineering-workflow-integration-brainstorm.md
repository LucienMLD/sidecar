---
date: 2026-02-19
topic: compound-engineering-workflow-integration
---

# Intégration des Workflows Compound-Engineering dans les Prompts Par Défaut de Sidecar

## What We're Building

Remplacer les prompts par défaut hardcodés dans `internal/plugins/workspace/prompts.go` (`DefaultPrompts()`) par des prompts qui intègrent les workflows compound-engineering (`/workflows:brainstorm`, `/workflows:plan`, `/workflows:work`).

Deux nouveaux prompts sont ajoutés ("Brainstorm Feature", "Plan Feature") pour accéder directement aux workflows. Le prompt "Begin Work on Ticket" est modifié pour que l'agent Claude propose `/workflows:brainstorm` si le ticket est ambigu, ou `/workflows:plan` si le ticket est suffisamment défini — avant de procéder à l'implémentation.

## Why This Approach

**Approche choisie : Modifier `DefaultPrompts()` dans le code Go**

Trois options ont été évaluées :

- **Modifier le code Go** (choisie) — centralise les defaults pour tous les users, synchronisé avec les tâches TD via `{{ticket}}`
- **Fichier `.sidecar/config.json`** — non-destructif mais ne change pas l'expérience par défaut
- **Code Go + documentation CLAUDE.md** — overhead sans valeur ajoutée claire

La modification directe des `DefaultPrompts()` est la solution la plus simple et la plus impactante. Elle garantit que tout nouvel utilisateur de Sidecar bénéficie des workflows compound-engineering dès le départ.

## Key Decisions

- **Granularité du choix brainstorm/plan** : Le prompt "Begin Work on Ticket" ne force pas un workflow spécifique — il demande à Claude d'évaluer si le ticket est bien défini et de proposer `/workflows:brainstorm` (ticket ambigu) ou `/workflows:plan` (ticket clair), puis de lancer `/workflows:work` pour l'exécution.

- **Synchronisation TD obligatoire** : Tous les prompts conservent le `td usage --new-session` au démarrage et utilisent le placeholder `{{ticket}}` pour maintenir le tracking des tâches.

- **Prompts existants conservés** : "Code Review Ticket" et "TD Review Session" restent essentiellement inchangés. Les prompts "Plan to Epic" sont fusionnés/alignés avec les workflows compound.

- **Structure des nouveaux prompts** :
  - `"Begin Work on Ticket"` (TicketRequired) → `td usage --new-session`, évalue si ticket ambigu → propose `/workflows:brainstorm` ou `/workflows:plan`, puis `/workflows:work {{ticket}}`
  - `"Brainstorm Feature"` (TicketNone) → `td usage --new-session`, lance `/workflows:brainstorm`
  - `"Plan Feature"` (TicketOptional) → `td usage --new-session`, lance `/workflows:plan`
  - `"Plan to Epic (No Impl)"` (TicketNone) → conservé, corps mis à jour pour utiliser `/workflows:plan` sans implémentation
  - `"Plan to Epic + Implement"` (TicketNone) → conservé, corps mis à jour pour `/workflows:plan` puis `/workflows:work`
  - `"Code Review Ticket"` → inchangé
  - `"TD Review Session"` → inchangé

- **Vérification runtime** : Sidecar vérifie la présence du plugin compound-engineering au moment de la sélection d'un prompt compound (Begin Work, Brainstorm Feature, Plan Feature, Plan to Epic). Chemin vérifié : `~/.claude/plugins/marketplaces/*/plugins/compound-engineering/commands/workflows/work.md` (glob). Si absent, affiche un message d'erreur **bloquant** avec les instructions d'installation (`claude plugins install compound-engineering`) — le workspace ne se lance pas. Les prompts non-compound (Code Review, TD Review) ne déclenchent pas cette vérification.

## Open Questions

Aucune — toutes les décisions ont été tranchées.

## Next Steps

→ `/workflows:plan` pour les détails d'implémentation
