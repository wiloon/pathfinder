# Pathfinder

An open-source AI-powered goal planner. Set any goal — finding a job, learning a skill, finishing a project — and Pathfinder generates a personalized daily plan, then adjusts it every day based on your real progress.

Built by people who were laid off and needed a smarter way to find the next opportunity.

## Features

- **Any goal, any context** — Describe your goal in text, or upload images (resume, job description screenshot, reference material). AI understands both.
- **Multi-goal, one plan** — Set a primary goal and optional secondary goals. AI merges them into a single daily plan, prioritizing tasks where goals overlap.
- **Daily task list with suggested time slots** — Each day you get a task list with recommended time blocks. You can reorder or adjust freely.
- **Event lifecycle management** — Insert a future event (text or screenshot). AI schedules preparation tasks before the event and prompts a retrospective after it, then re-plans accordingly.
- **Daily check-in** — Each evening: what did you complete, what blocked you, where do you want to focus tomorrow?
- **Dynamic re-planning** — AI adjusts tomorrow's plan based on tonight's check-in.
- **No account required** — Data stored locally in your browser. Export/import as JSON for backup or device transfer.
- **Multi-goal management** — Add, remove, or reprioritize goals at any time. Switch which goal is primary whenever you need to.

## Tech Stack

**Backend**
- Go + Gin
- SQLite (via gorm)
- MiniMax AI API
- Session authentication (httpOnly cookie)

**Frontend**
- Next.js 15 (App Router)
- React 19.2
- Tailwind CSS + shadcn/ui
- TanStack Query + react-hook-form + Zod

**Deployment**
- Docker / Podman
- Single container or docker-compose

## Project Status

🚧 Early development — MVP in progress

## Roadmap

### Phase 1 — MVP (local storage, no login)
- [ ] Onboarding: set primary goal + optional secondary goals (text + image upload)
- [ ] AI goal analysis: extract background, identify goal overlaps, generate initial plan
- [ ] Daily view: task list with suggested time slots (reorderable)
- [ ] Event lifecycle: insert event → AI schedules prep tasks before + retrospective after → re-plan
- [ ] Evening check-in: completed / blocked / tomorrow's focus
- [ ] AI re-planning based on check-in
- [ ] Local storage persistence (IndexedDB) + JSON export/import
- [ ] Goal management: add/remove goals, change primary goal at any time

### Phase 2
- [ ] User accounts + cloud sync
- [ ] Weekly plan view and full roadmap view
- [ ] Multi-device support

### Phase 3
- [ ] Web3 mode: decentralized storage (IPFS/Arweave) + wallet-based identity as an opt-in alternative

## Getting Started

> 🚧 Setup instructions will be added once the initial code is committed.
>
> Prerequisites: Go 1.26+, Node.js 24+, pnpm

## Contributing

Contributions welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

If you were also laid off and want to work on this together — open an issue or reach out directly.

## License

MIT
