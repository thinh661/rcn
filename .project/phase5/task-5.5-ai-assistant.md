# Task 5.5: AI Assistant

## Frontend (agy) — main work
1. Component: `components/Notebooks/AIAssistant.tsx`
2. Chat panel bên cạnh notebook
3. Gọi backend API để gửi prompt + context
4. Hiển thị code suggestions, explanations

## Backend (main)
1. Service: `services/ai_assistant.go`:
   - `POST /api/v1/ai/ask` — gửi prompt (code context + question)
   - Proxy tới OpenAI API hoặc local LLM (Ollama)
   - Trả về streaming response
2. Config: `AI_ENABLED`, `AI_PROVIDER`, `AI_API_KEY`, `AI_MODEL`
