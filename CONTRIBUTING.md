# Contributing to Zenc Server

Thank you for your interest in contributing to **Zenc Server**! We welcome bug reports, feature requests, documentation improvements, and code contributions from the community.

---

## Code of Conduct

We are committed to providing a welcoming, inclusive, and harassment-free environment for all contributors. Please be respectful, constructive, and collaborative in all interactions.

---

## How Can I Contribute?

### 1. Reporting Bugs
- Check existing [GitHub Issues](https://github.com/cyper98/zenc-server/issues) before opening a new issue.
- Include a clear title, description, steps to reproduce, expected vs. actual behavior, and environment details (Go version, OS, Redis version).
- Provide logs or minimal reproducible examples whenever possible.

### 2. Suggesting Enhancements
- Open a GitHub Issue using the Feature Request template.
- Explain the problem, the proposed solution, and why it benefits the broader community.

### 3. Submitting Code (Pull Requests)
- For major architectural changes, discuss them in an issue before writing code.
- Small bug fixes and documentation improvements can be submitted directly as PRs.

---

## Development Workflow

### Prerequisites
- **Go 1.22+** (Go 1.23 recommended)
- **Redis 7.0+**
- **golangci-lint** (optional but recommended)
- **Docker & Docker Compose**

### Setup Local Environment
1. Fork the repository on GitHub and clone your fork:
   ```bash
   git clone https://github.com/<your-username>/zenc-server.git
   cd zenc-server
   ```
2. Download dependencies:
   ```bash
   make deps
   ```
3. Start Redis in the background:
   ```bash
   docker run -d --name zenc-redis -p 6379:6379 redis:7-alpine
   ```
4. Build and run locally:
   ```bash
   make build
   make run
   ```

### Branching Convention
Create a branch with a descriptive name from `main`:
- `feat/feature-name` for new features
- `fix/bug-description` for bug fixes
- `docs/topic-name` for documentation
- `perf/optimization-name` for performance improvements

---

## Coding Standards & Testing

### Code Quality
- Follow standard Go idioms and [Effective Go](https://go.dev/doc/effective_go).
- Format your code before committing:
  ```bash
  make fmt
  ```
- Run the linter:
  ```bash
  make lint
  ```

### Running Tests
All tests must pass before submitting a Pull Request:
```bash
make test
```
To check test coverage:
```bash
make test-coverage
```

### Commit Messages
We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:
- `feat: add adaptive layer filtering in SFU`
- `fix: resolve race condition in peer track subscription`
- `docs: update deployment guidelines`
- `refactor: simplify room participant cleanup`

---

## Contributor License Agreement (DCO)

To ensure that contributions can be safely distributed under the project's **Apache License 2.0**, all contributors must sign off their commits certifying adherence to the **Developer Certificate of Origin (DCO)**.

Add the `-s` flag to sign off your commit:
```bash
git commit -s -m "feat: your commit message"
```
This appends a line:
```
Signed-off-by: Your Name <your.email@example.com>
```
By submitting a Pull Request, you agree that your contributions will be licensed under the project's **Apache License 2.0**.

