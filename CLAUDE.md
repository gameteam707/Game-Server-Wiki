# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Purpose

This is a **documentation-only wiki** for the KFC Project — an online gaming platform backend system. It contains no source code, build system, or tests. The primary content is the `junior-developer-guide/` series: a 16-part onboarding guide (written in Traditional Chinese) for junior developers joining the team.

## Content Structure

The guide is organized in numbered Markdown files (`00` through `15`) following a deliberate reading order:

1. **Global view** (00-03): Project overview, Clean Architecture philosophy, directory structure
2. **Layer deep-dives** (04-07): Domain → UseCase → Adapter → Infrastructure (inside-out)
3. **Cross-cutting concerns** (08-11): DI, API design/middleware, database/repository, microservice communication
4. **Operations** (12-14): Observability, deployment/containerization, testing strategy
5. **Growth path** (15): Learning roadmap for junior developers

## What This Wiki Documents

The KFC platform consists of multiple Go microservices:

| Service | Role | Protocol |
|---------|------|----------|
| **admin-backend** | Admin portal (CRUD, RBAC, audit) | Gin HTTP + Vue 3 frontend |
| **config-service** | Config sync hub | HTTP + SSE (MongoDB Change Streams) |
| **sfc-stream-game** | Game engine (gameservice + gameapi + connector) | gRPC, HTTP, WebSocket |

Supporting modules: `math-lib`, `go-observability`, `kfc-k8s`, `Special_Game_Pipeline`.

Tech stack: Go 1.25+, Gin, gRPC, MongoDB, Redis, Kafka, OpenTelemetry, Docker, Kubernetes.

## Writing Guidelines

- All documentation is in **Traditional Chinese** (繁體中文)
- Each guide file uses a consistent pattern: concept explanation → code examples → Q&A for juniors
- Code examples reference the actual project directory structure (e.g., `internal/domain/entity/`, `internal/usecase/{feature}/`)
- The architecture described follows Clean Architecture / Hexagonal Architecture with strict dependency rules: Infrastructure → Adapter → UseCase → Domain

## Training System

This repo includes an agent-driven training system for junior developers.

### Key Commands
- `/init` — Initialize a trainee's environment (progress folder, branch, practice project)
- `/train` — Main training interaction (quiz, implementation tasks, review)

### Training Structure
- `training/curriculum.yaml` — Course definition (8 stages, required/optional)
- `training/questions/*.yaml` — Core quiz questions per chapter
- `training/checkpoints.yaml` — Mentor review checkpoint configuration
- `training/scaffold/` — Practice project template (Go + Gin)
- `trainees/<name>/` — Per-trainee progress, answers, and reviews (on trainee branches)

### Agent Behavior
- The training agent follows rules in `.claude/rules/training-context.md`
- Never gives direct answers to quiz questions — uses hints and guided questioning
- Never provides complete implementation code — gives directional feedback
- Progress is tracked in `progress.yaml` and committed to the trainee's branch
