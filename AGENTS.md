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
