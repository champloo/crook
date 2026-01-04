<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

## Project Conventions

Make sure to review @openspec/project.md for project conventions.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Clean up** - Clear stashes, prune remote branches
5. **Verify** - All changes committed 
6. **Hand off** - Provide context for next session

### Task tracking

* Use 'bd' CLI command for any task tracking work. Do not use the beads MCP.
* When creating tasks make sure that tasks are detailed focusing carefully on dependencies, detailed designs and potential parallelization.
  * Put enough detail and context in the Beads tasks to enable you to execute them at a later time
  * Add references relevan openspec specs, design and proposals to allow you to fetch data later on
* When I ask you to work on multiple tasks do tasks one by
  * Provide a summary for each task as soon as you finish it and then commit the changes to git.
  * Don't stop until you have completed all tasks
  * Finally provide overall summary for all tasks
