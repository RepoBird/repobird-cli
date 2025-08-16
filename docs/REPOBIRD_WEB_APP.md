# RepoBird - Remote AI Coding Agents

## Overview

**One-shot issue to PR - just run the agent and get production-ready code.**

RepoBird is a SaaS platform that provides AI-powered software engineering assistance through GitHub integration. Our platform transforms how development teams handle their backlog by autonomously creating pull requests and solving software issues based on simple user commands. RepoBird integrates seamlessly with GitHub, allowing developers to trigger AI agents directly from issue comments or through our web dashboard.

## What is RepoBird?

RepoBird is an AI-powered engineering automation platform that delivers complete engineering solutions, not just code snippets. Our agents:

- **Generate comprehensive implementation plans** before writing code, providing detailed strategies you can review and refine with your team
- **Analyze your entire codebase** to understand dependencies and follow existing patterns
- **Create production-ready pull requests** with proper commits, comprehensive tests, and detailed explanations
- **Work like senior engineers** who understand your repository's context, history, and conventions
- **Follow engineering best practices** including atomic commits, error handling, and clean architecture
- **Powered by state-of-the-art AI** - the industry's most advanced coding agents for superior code quality and understanding

## Top 3 Selling Points

1. **Best-in-Class AI Agent** - We use the industry's most advanced AI coding agents. Get superior code quality, better understanding of complex requirements, and production-ready implementations that follow best practices and your codebase patterns.

2. **Cloud-Powered Execution** - Each agent runs in its own isolated cloud environment. Launch multiple agents simultaneously across different issues without file conflicts or resource constraints. Your team stays productive while agents work in the background.

3. **Native GitHub Integration** - Fully automated Git handling ensures clean, atomic commits with descriptive messages. Every PR includes comprehensive summaries and detailed change explanations. Review code that looks like it was written by a senior engineer.

## Additional Key Value Propositions

4. **Enterprise-Grade Development Environment**: State-of-the-art sandboxed VM with complete development ecosystem including multi-language support, development tools, database clients, and unlimited package access.

## How It Works

RepoBird follows a smart workflow that ensures high-quality results:

1. **Plan First** (Optional but Recommended): Request a comprehensive implementation plan that you can review with your team
2. **Execute with Confidence**: Once aligned, trigger the agent to implement the solution
3. **Iterate Seamlessly**: Request changes directly in PR comments for instant updates

**Trigger Methods:**
- **GitHub Comments**: Mention `@RepoBirdBot` in any issue or PR with instructions
- **Web Dashboard**: Launch agent runs directly from the RepoBird UI without GitHub comments

For more detailed information about RepoBird's features, architecture, and implementation details, explore the other documentation files in the `/docs` directory.

## Capabilities
- **Implementation Planning**
  Use `@RepobirdBot plan` on an **Issue** to generate a detailed implementation strategy as a GitHub comment. This allows teams to review and align on approach before code generation.
- **Automated Pull‑Request Creation**
  Mention `@RepobirdBot run …` on an **Issue** to spin up an agent that writes code in a new branch and opens a pull request.
- **Smart Workflow: Plan → Review → Run**
  The recommended approach: First use `plan` to understand the implementation, review with your team, then use `run` for aligned execution. This provides predictable outcomes and streamlined reviews.
- **Pull‑Request Updates**
  Inside an **existing PR**, comment with instructions (e.g. `@RepobirdBot run Refactor to hooks`) and the agent pushes new commits to the same branch.
- **Branch Targeting (Issues Only)**
  Supply `source:<branch>` / `target:<branch>` options to control where the work starts and where the PR is opened.
- **UI Trigger Mode**
  Launch agent runs directly from the RepoBird dashboard without GitHub comments. Simply navigate to your repository page and click "Run" to create PRs through the web interface. Note: GitHub App installation is still required for repository access.
- **State-of-the-Art AI Environment**
  Enterprise-grade sandboxed VM with pre-configured languages (Python, Node.js, Ruby, C/C++), development tools (Git, Docker, build systems), database clients, and unlimited package access via APT.

## Benefits
- **Accelerated Delivery** – Turn backlog items into merge‑ready pull requests without dropping your current task.
- **Hands‑Free Iteration** – Request follow‑up changes in the PR thread; RepoBird edits the branch automatically.
- **Reduced Context Switching** – No external dashboards or API setup needed; everything happens inside Issues and PRs you already use.
- **Ship 100x Faster** – Parallel AI agents generating PRs with zero conflicts, powered by isolated cloud environments.

## Getting Started

| Requirement     | Details                                                                 |
| --------------- | ----------------------------------------------------------------------- |
| **Plans**       | Free: 3 runs/month   ·  Pro: 30 runs/month                              |
| **Permissions** | Issues, Pull Requests, Contents (Read/Write), Comments (Read/Write)     |
| **Availability**| Public **Beta**                                                         |
| **Environment** | Full development ecosystem with 59,000+ packages available              |

**Setup**

1.  Install the **RepoBird** GitHub App on the repositories you want it to access. You can find the installation link on the main page or your dashboard if not connected.
2.  Open a GitHub Issue and start with a plan by mentioning the bot:
    ```markdown
    @RepobirdBot plan Analyze the requirements and suggest implementation approach
    ```
3.  Review the generated plan with your team, then execute:
    ```markdown
    @RepobirdBot run source:main target:feat/setup-auth Implement JWT authentication based on the plan above
    ```
    For help with available commands:
    ```markdown
    @RepobirdBot help
    ```
    *(See the [Repobird.ai/docs](https://repobird.ai/docs) page for more command details and options.)*

## Product Roadmap Highlights

### Recently Completed ✅
- **Claude Code Integration**: Superior AI agent for code generation
- **UI-Triggered Runs**: Launch agents directly from dashboard
- **Internal Notifications**: Customizable email preferences for run updates
- **State-of-the-Art AI Environment**: Enterprise-grade sandboxed VM with full dev ecosystem

### Coming Soon 🚧
- **Credit-Based System** (Q1 2025): Flexible pricing based on actual usage
- **CLI Tool** (Q2 2025): Command-line interface for power users
- **API Triggers** (Q2 2025): RESTful API for programmatic control
- **Project Memory** (Q2 2025): Persistent context across multiple runs
- **AI Auto-Suggest** (Q2 2025): Intelligent follow-up suggestions
- **VSCode Extension** (Q3 2025): Native IDE integration

For the complete roadmap, see our [Product Roadmap](/docs/business/PRODUCT_ROADMAP.MD).
