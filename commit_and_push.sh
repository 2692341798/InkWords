#!/bin/bash
git add .
git commit -m "feat: upgrade to DeepSeek V4 and add Git subDir support

- Removed custom Ollama/Redis semantic caching
- Refactored prompt structure to fully utilize DeepSeek V4 Native Prompt Caching
- Upgraded model to deepseek-v4-flash with 1M context (2M chars limit) and 128k output
- Added sub_dir parameter to GitFetcher using sparse-checkout for large repos
- Updated frontend GitSourceInput to support SubDir
- Updated all project documents to reflect the latest architecture and rules" > .git_commit.log 2>&1

git tag v2.6.0 >> .git_commit.log 2>&1
git push origin main >> .git_commit.log 2>&1
git push origin v2.6.0 >> .git_commit.log 2>&1
