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

https://github.com/kcolemangt/llm-router/assets/20099734/7220a3ac-11c5-4c89-984a-29d1ea850d10

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

## MacOS Permissions

When attempting to run LLM-router on MacOS, you may encounter permissions errors due to MacOS's Gatekeeper security feature. Here are several methods to resolve these issues and successfully launch the application.

### Method 1: Modify Permissions via Terminal
If you receive a warning about permissions after downloading the release binary, change the file's permissions to make it executable:

```sh
chmod +x llm-router-darwin-arm64
./llm-router-darwin-arm64
```

### Method 2: Use the `spctl` Command
If the above does not resolve the issue and you still face security pop-ups:

1. Add the application to the allowed list using:
   ```sh
   sudo spctl --add llm-router-darwin-arm64
   ```

2. Attempt to run the application again:
   ```sh
   ./llm-router-darwin-arm64
   ```

### Method 3: Open Directly from Finder
For issues persisting beyond previous steps:

1. Find `llm-router-darwin-arm64` in Finder.
2. Control-click on the app icon and select 'Open' from the context menu.
3. In the dialog that appears, click 'Open'. Admin users may need to authenticate.

This step should register the application as a trusted entity on your Mac, bypassing Gatekeeper on subsequent launches.

### Method 4: Manual Override in System Preferences
Should the above methods fail:

1. Open System Preferences and navigate to Security & Privacy.
2. Under the 'General' tab, you may see an 'Allow Anyway' button next to a message about LLM-router.
3. Click 'Allow Anyway' and try running the application again.

### Method 5: Build from Source
If none of the above methods work, consider building the application from source:

1. Download the source code.
2. Ensure you have a current version of [Go](https://go.dev) installed.
3. Build the application:
   ```sh
   make
   ./build/llm-router-local
   ```

## Connect

* X (twitter) [@kcolemangt](https://x.com/kcolemangt)

* LinkedIn [Keith](https://www.linkedin.com/in/keithcoleman/)
