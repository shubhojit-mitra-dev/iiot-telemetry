# AI Agent Instructions and Coding Standards

## 1. Commit Strategy & Version Control (CRITICAL)
- **Atomic Commits:** Every single logical change must be committed separately. Do not bundle multiple unrelated changes into a single commit.
- **File Limit:** At maximum, **only 2 files are allowed per commit**. NO EXCEPTIONS. If a feature requires editing 3 files, break it down into multiple commits.
- **Branching:** You must create a new branch for any feature, fix, or phase (e.g., `git checkout -b feature/phase-1-ingestion`).
- **Pull Requests (PRs):** Once the code for a phase or feature is complete, push the branch to the remote repository.
- **Review Process:** You must ask the Senior Engineer (User) to manually review the PR. The User will read everything, comment, and perform the merge manually if the code meets standards. **DO NOT merge branches yourself.**

## 2. Code Quality & Architecture
- Write clean, production-grade code with appropriate error handling.
- Follow the specific architectural instructions laid out in the `docs/implementations/` folder.
- Maintain thread safety and handle concurrency properly, especially in Go.
- Include comments explaining *why* complex decisions were made, not just *what* the code does.
- **Idiomatic Go:** Follow proper code standards. Since the project is in Go, write idiomatic Go, adhering to "Effective Go" guidelines and language specifications as intended by the language designers.

## 3. Testing Standards & Test-Driven Development (TDD) (CRITICAL)
- **FIRST Principle:** All tests must strictly adhere to the FIRST principles:
  - **F - Fast:** Unit tests must execute in milliseconds without external network or persistent storage dependencies.
  - **I - Independent:** Tests must not depend on execution order or shared mutable global state.
  - **R - Repeatable:** Tests must produce identical results across local environments, Docker containers, and CI runners.
  - **S - Self-Validating:** Tests must have clear, automated assertions with zero manual inspection needed.
  - **T - Timely:** Tests must be written before or alongside code.
- **Test-First Discipline:** No business logic is written unless proper test cases are ready for it. Write tests first, then write the minimum code required to pass those tests. Never write code that does more than what the test suite demands.
- **Exhaustive Edge Cases:** Cover boundaries, nil/empty payloads, negative values, channel buffer saturation, concurrent race conditions (`go test -race`), slow client drops, and panic recoveries.
- **Strict 80%+ Test Coverage Threshold:** Backend packages must maintain at least **80% statement test coverage**. Any PR or commit dropping coverage below 80% is unacceptable and must not be merged.
- **DevSecOps CI Mandate:** Automated CI must execute on every push and PR, enforcing:
  1. Unit and integration tests with the Go race detector (`-race`).
  2. Automatic coverage calculation and strict failure if coverage < 80%.
  3. Security scanning with static analysis tools (`govulncheck`, `gosec`).
  4. Code formatting and linting validation (`golangci-lint`).
  5. Frontend typecheck and build validation (`tsc && vite build`).
