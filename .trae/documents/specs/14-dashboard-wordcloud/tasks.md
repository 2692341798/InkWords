# 0. Background
Replace the overcrowded PieChart in the Dashboard with a beautiful, dynamic Word Cloud, filtering to Top 20 tech stacks.

# 1. Implementation Steps
- Modified `backend/internal/api/user.go`: Added logic in `GetUserStats` to sort `techStackStats` by `Count` descending, and slice to max 20 elements. This reduces network payload and guarantees top data.
- Modified `frontend/package.json`: Added `"react-wordcloud": "^1.2.7"`.
- Modified `frontend/Dockerfile`: Changed `npm ci` to `npm install --legacy-peer-deps` to handle new dependencies smoothly without a regenerated package-lock.json and avoid React 19 peer-dep errors.
- Modified `frontend/src/components/Dashboard.tsx`: Removed Recharts PieChart components, imported `ReactWordcloud`, and mapped `tech_stack_stats` to `{ text, value }` structure with custom font styling and rotations.

# 2. Testing
The user needs to manually run `docker compose down && docker compose up -d --build` due to terminal unavailability, and check the dashboard on `http://localhost`.