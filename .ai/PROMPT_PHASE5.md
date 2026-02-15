# PROMPT_PHASE5.md — GitHub Pages landing site for hue-cli (docs/ + screenshots + badges)

You are an autonomous coding agent working in this repo. Implement Phase 5: create a **nice landing/project page** hosted via **GitHub Pages** (no custom domain) and add screenshots + badges.

Follow repo rules in `AGENTS.md` and `.ai/AGENTS.md`:
- Use `jj` for VCS.
- Make local commits for meaningful milestones.
- Do NOT push to origin unless I explicitly ask.

## Goal
Create a simple, attractive static landing page for `huectl` and wire it for GitHub Pages publishing from the `docs/` folder on `main`. GitHub Pages supports publishing from a branch/folder like `/docs`. :contentReference[oaicite:0]{index=0}

## Requirements

### A) Site location & structure
- Create a `docs/` folder with:
  - `docs/index.html`
  - `docs/styles.css`
  - `docs/assets/` (images/svgs/screenshots)
- Keep it framework-free (plain HTML/CSS) for reliability.

### B) Page content (landing page)
The landing page should include:

1) **Hero section**
- Name: `huectl`
- 1–2 sentence value prop: “Fast, scriptable Philips Hue CLI (Hue API v2).”
- Primary CTA buttons:
  - “Install” (scroll to install section)
  - “GitHub” (link to repo)

2) **Badges row** (SVG badges)
Include at least these badges using Shields.io (embed as `<img>`):
- GitHub Actions CI status (for your CI workflow)
- Latest release (once you start tagging)
- License (if present)
- Go version (optional as static badge)

Use standard Shields.io badges and keep them clickable (wrap in `<a>`). Shields.io is the canonical badge service. :contentReference[oaicite:1]{index=1}

3) **Features section**
- Hue API v2
- Fast CSV output for scripting
- No bridge discovery required (explicit config)
- macOS + Linux

4) **Screenshots section**
Add a “Screenshots” section showing 2–3 screenshots:
- `huectl lights list --csv`
- A “toggle” example
- (Optional) “wide”/group output if implemented
If real screenshots aren’t available in the repo yet:
- Generate placeholder screenshots by running the commands locally is NOT allowed here;
  instead, create “terminal-style” screenshots in a simple way:
  - Render them as images via HTML/CSS (fake terminal blocks) OR
  - Add them as `<pre>` blocks styled as a terminal, and only add image screenshots later.
Prefer: terminal-styled `<pre>` blocks plus optional “image slot” placeholders.

5) **Install section**
Document 3 install options:
- Homebrew tap (once set up)
- GitHub Releases manual download
- `install.sh` script
Also clearly note: GitHub Releases-based install requires that at least one release exists.

6) **Quickstart section**
Show examples:
- `huectl lights list --csv`
- `huectl lights toggle --id ...`
- env vars:
  - `HUE_BRIDGE_HOST`
  - `HUE_API_TOKEN`

7) **Footer**
- Links: GitHub repo, issues, releases

### C) Repo docs updates
- Update root `README.md` to include:
  - link to the GitHub Pages site URL pattern: `https://<user>.github.io/<repo>/`
  - a short note “Project website is hosted via GitHub Pages.”
(Do not hardcode the final URL if uncertain; derive owner/repo from git remote if possible and document if placeholder.)

### D) GitHub Pages configuration guidance
You cannot change repo settings via code, but you MUST add a short setup guide:
- Create `docs/PAGES.md` (or add a section to `docs/index.html` commented) with steps:
  - Repo Settings → Pages → “Deploy from a branch” → Branch `main` → Folder `/docs` → Save.
GitHub docs confirm Pages can publish from `/docs` on the source branch. :contentReference[oaicite:2]{index=2}

### E) Quality / design constraints
- Make it look modern but lightweight:
  - Good typography, spacing, simple palette, responsive layout.
  - Mobile-friendly.
  - No external JS dependencies.
- Use only local assets in `docs/assets/`.

### F) Commits
Make 2–3 logical `jj` commits, e.g.:
- `docs: add GitHub Pages landing site`
- `docs: add badges and installation section`
- `docs: link site from README`

Do not push to origin.

## Done criteria
- `docs/index.html` renders nicely.
- README links to the site.
- Clear instructions for enabling GitHub Pages from `/docs`.
- No secrets or private info included.
- Existing Go build/test workflows remain unchanged.

Start now.
