---
name: Bug Report
about: Create a report to help us improve llm-router
title: "[BUG]"
labels: bug
assignees: ''
---

**Is this a Bug or a Question?**

Thank you for contributing! Before filing a bug report, please consider the following:

1.  **Is this a question about how to use `llm-router`, configure a specific backend, or understand its behavior?**
    *   If yes, please ask your question in our **[GitHub Discussions Q&A section](https://github.com/kcolemangt/llm-router/discussions/categories/q-a)** instead. This is the best place for community support and usage help.

2.  **Have you checked the [README](https://github.com/kcolemangt/llm-router/blob/main/README.md) and existing [Discussions](https://github.com/kcolemangt/llm-router/discussions)?**
    *   Your question might already be answered there.

3.  **Are you sure this is a software defect (a bug) in `llm-router` itself?**
    *   Examples: A crash, unexpected error message (not related to configuration), incorrect routing logic, failure to respect configuration settings.

**If you are confident this is a bug in `llm-router`, please proceed with filling out the details below.** Otherwise, please use the Discussions Q&A.

---

**Describe the bug**
A clear and concise description of what the bug is.

**Steps to Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '...'
3. Scroll down to '...'
4. See error

**Expected Behavior**
A clear and concise description of what you expected to happen.

**Screenshots**
If applicable, add screenshots to help explain your problem.
> **⚠️ Important Security Notice:** Before attaching screenshots, logs, or configuration files, please ensure that they are **thoroughly sanitized**. 
> **Do NOT include any sensitive information** like API keys, access tokens, private URLs, or personal data. 
> Failure to sanitize your attachments could expose sensitive information.

**Environment (please complete the following information):**
 - OS: [e.g. Windows, macOS, Linux]
 - `llm-router` Version [e.g. 1.0.0]
 - How you installed `llm-router` (e.g., `go install`, Docker, built from source)
 - If using Docker, the Docker image tag

**Configuration (`config.json` - SANITIZED)**
Please paste the relevant parts of your `config.json` here. **Remember to remove any API keys or sensitive URLs.**

```json
{
  // Paste your SANITIZED config here
}
```

**Logs (SANITIZED)**
If applicable, provide relevant logs. **Remember to remove or redact any sensitive information.**

```
Paste your SANITIZED logs here
```

**Additional Context**
Add any other context about the problem here.

---

### 🔒 Sanitization Checklist
*Before submitting, please confirm the following:*

- [ ] I have carefully reviewed my report, including attachments (screenshots, logs, config excerpts).
- [ ] I have removed or redacted **ALL** API keys, access tokens, or other credentials.
- [ ] I have removed or redacted **ALL** private URLs (like ngrok tunnels).
- [ ] I have verified that no other sensitive personal or organizational information is included. 