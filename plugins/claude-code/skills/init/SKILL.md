---
description: >
  Initialize a new SDD (Spec-Driven Development) project with Hoofy.
  Creates the project structure and guides you through the proposal stage.
  Use when starting a new project, bootstrapping, or saying "start SDD".
---

# Initialize SDD Project

You are starting a new Spec-Driven Development project using Hoofy.

## What to do

1. Parse the user's input from `$ARGUMENTS` to extract a project name and description. If not provided, ask the user.

2. Call `mcp__hoofy__sdd_init_project` with:
   - `name`: the project name
   - `description`: brief project description
   - `mode`: ask the user if they prefer `guided` (step-by-step, recommended for first time) or `expert` (streamlined)

3. After initialization, guide the user to create their **proposal** by calling `mcp__hoofy__sdd_create_proposal`. Ask them about:
   - What problem does this project solve?
   - Who are the target users?
   - What's the proposed solution (high-level, no tech details)?
   - What's explicitly out of scope?
   - What does success look like?

4. After the proposal, check status with `mcp__hoofy__sdd_get_context` and tell the user the next step in the pipeline.

## Important

- The SDD pipeline is sequential: init → propose → specify → clarify → design → tasks → validate
- Each stage builds on the previous one
- The Clarity Gate at the `clarify` stage blocks progress until requirements are clear enough
- Do NOT skip stages
