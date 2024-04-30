# LLM-router

## Overview

Access models from OpenAI, Groq, local Ollama, and other providers by setting LLM-router as the base URL in Cursor. 

LLM-router is a reverse proxy that routes `chat/completions` API requests to various OpenAI-compatible backends based on the model's prefix.

## Background

[Cursor](https://cursor.sh) lacks support for local models and the flexibility to switch between multiple LLM providers efficiently. There's clear demand in the community, evidenced by months of unanswered requests for such features on Cursor's GitHub and Discord channels.

The specific challenges with Cursor include:
1. The `Override OpenAI Base URL` setting requires a URL that is publicly accessible and secured with `https`, complicating the use of local models.
2. The platform allows configuration for only one LLM provider at a time, which makes it cumbersome to switch between service providers.

LLM-router overcomes these limitations, allowing seamless switching between locally served models like Ollama and external services such as OpenAI and Groq.

<video controls loop autoplay muted>
  <source src="assets/llm-router-cursor-preview-web.mp4" type="video/mp4">
  <img src="assets/llm-router-preview.gif" alt="Fallback animation of Ollama in Cursor via LLM-router">
</video>

## Getting Started

1. Launch LLM-router to manage API requests across multiple backends:
```sh
OPENAI_API_KEY=<YOUR_OPENAI_KEY> ./llm-router-darwin-arm64
```

2. Launch [ngrok](https://ngrok.com) to create a public HTTPS endpoint for your LLM-router:
```sh
ngrok http 11411
```

Configure the `Override OpenAI Base URL` in Cursor's model settings to point to your ngrok address appended with `/v1`:
```
https://xxxx.ngrok-free.app/v1
```

Define your preferred models:
```
ollama/phi3
ollama/llama3:70b
```

## Details

Routes `chat/completions` API requests to any OpenAI-compatible LLM backend based on the model's prefix. Streaming is supported.

LLM-router can be set up to use individual API keys for each backend, or no key for services like local Ollama.

By default, requests to LLM-router are secured with your `OPENAI_API_KEY` as Cursor already includes this with every request.

Here is an example of how to configure Groq, Ollama, and OpenAI backends in `config.json`:
```json
{
	"listening_port": 11411,
	"backends": [
		{
			"name": "openai",
			"base_url": "https://api.openai.com",
			"prefix": "openai/",
			"default": true,
			"require_api_key": true
		},
		{
			"name": "ollama",
			"base_url": "http://localhost:11434",
			"prefix": "ollama/"
		},
		{
			"name": "groq",
			"base_url": "https://api.groq.com/openai",
			"prefix": "groq/",
			"require_api_key": true,
			"key_env_var": "GROQ_API_KEY"
		}
	]
}
```

In this configuration, OpenAI serves as the default backend, allowing you to use model identifiers like `openai/gpt-4-turbo` or simply `gpt-4-turbo`. Models on Ollama and Groq, however, must be prefixed with `ollama/` and `groq/` respectively.

Provide the necessary API keys via environment variables:
```sh
OPENAI_API_KEY=<YOUR_OPENAI_KEY> GROQ_API_KEY=<YOUR_GROQ_KEY> ./llm-router-darwin-arm64
```

## Connect

* X (twitter) [@kcolemangt](https://x.com/kcolemangt)

* LinkedIn [Keith](https://www.linkedin.com/in/keithcoleman/)