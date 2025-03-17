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

1. Create your configuration file:
   - A sample configuration file `config-sample.json` is provided in the repository
   - Either rename this file to `config.json` or create a new `config.json` file using the example below
   - This file defines which LLM backends are available and how to route to them

2. Launch LLM-router to manage API requests across multiple backends:
```sh
./llm-router-darwin-arm64
```
   - If you haven't set a `LLMROUTER_API_KEY` environment variable, the program will generate a random strong API key
   - **Copy this API key** as you'll need it for Cursor's configuration

3. Launch [ngrok](https://ngrok.com) to create a public HTTPS endpoint for your LLM-router:
```sh
ngrok http 11411
```
   - Take note of the HTTPS URL provided by ngrok (e.g., `https://xxxx.ngrok-free.app`)

4. Configure Cursor to use your LLM-router:
   - Open Cursor's settings and go to the "Models" section
   - Paste the LLM-router API key (from step 2) into the "OpenAI API Key" field
   - Click the dropdown beneath the API key field labeled "Override OpenAI Base URL (when using key)"
   - Enter your ngrok URL (from step 3) in this field
   - Click the "Save" button next to this field
   - Click the "Verify" button next to the API key field to confirm the connection

5. Define your preferred models in Cursor using the appropriate prefixes:
```
ollama/phi4
openai/gpt-4o-mini
groq/deepseek-r1-distill-llama-70b-specdec
```

⚠️ **Important Warning**: When clicking "Verify", Cursor randomly selects one of your enabled models to test the connection. Make sure to **uncheck any models** in Cursor's model list that aren't provided by the backends configured in your `config.json`. Otherwise, verification may fail if Cursor tries to test a model that's not available through your LLM-router.

## Configuration

Here is an example of how to configure Groq, Ollama, and OpenAI backends in `config.json`:
```json
{
	"listening_port": 11411,
	"llmrouter_api_key_env": "LLMROUTER_API_KEY",
	"aliases": {
		"o1": "groq/deepseek-r1-distill-qwen-32b",
		"o3-mini": "ollama/qwq"
	},
	"backends": [
		{
			"name": "openai",
			"base_url": "https://api.openai.com/v1",
			"prefix": "openai/",
			"default": true,
			"require_api_key": true,
			"key_env_var": "OPENAI_API_KEY"
		},
		{
			"name": "ollama",
			"base_url": "http://localhost:11434/v1",
			"prefix": "ollama/",
			"role_rewrites": {
				"developer": "system"
			}
		},
		{
			"name": "groq",
			"base_url": "https://api.groq.com/openai/v1",
			"prefix": "groq/",
			"require_api_key": true,
			"key_env_var": "GROQ_API_KEY",
			"role_rewrites": {
				"developer": "system"
			}
		}
	]
}

```

In this configuration, OpenAI serves as the default backend, allowing you to use model identifiers like `openai/gpt-4o-mini` or simply `gpt-4o-mini`. Models on Ollama and Groq, however, must be prefixed with `ollama/` and `groq/` respectively. This configuration also causes Cursor to send optimized reasoning prompts to Groq's `deepseek-r1-distill-llama-70b-specdec` model and the local Ollama `qwq` model.

### Optimizing for Reasoning Models and Prompt Techniques

Clients such as Cursor send specialized prompts to specific models recognized for enhanced reasoning performance. Usually, these optimized prompts target proprietary models. The **Model Aliases** and **Role Rewrites** features in LLM-router allow you to extend these optimizations to reasoning-oriented models hosted locally (such as via Ollama) or by alternative providers (like Groq).

By combining aliases and role rewrites, you can route optimized reasoning prompts effectively across different backend models.

#### Model Aliases

Aliases allow mapping specific client-recognized reasoning model identifiers to backend models of your choice. This causes Cursor and similar clients to use specialized reasoning prompts designed for those models.

Simple example:

```json
"aliases": {
    "o1": "groq/deepseek-r1-distill-llama-70b-specdec"
}
```

In this example:
- Cursor sends prompts optimized for OpenAI's `o1`, but LLM-router redirects these requests to Groq's reasoning-focused model `deepseek-r1-distill-llama-70b-specdec`.
- The Groq backend benefits from Cursor's specialized reasoning prompts originally intended for `gpt-4-turbo`.

#### Role Rewrites

Role rewrites ensure that message roles from clients are correctly translated to match backend provider expectations. Clients using specialized reasoning prompts often use custom roles (e.g., `developer`) that may not be recognized universally.

Simple example:

```json
"role_rewrites": {
    "developer": "system"
}
```

- Messages with the role `developer` from Cursor will be rewritten to the standard role `system`, ensuring compatibility with backends.

#### Unsupported Parameters

Each model provider may support different parameters in their API requests. The `unsupported_params` option allows you to specify parameters that should be automatically removed from requests before forwarding them to specific backends, preventing errors due to incompatible parameters.

Simple example:

```json
"unsupported_params": [
    "reasoning_effort"
]
```

In this example:
- The parameter `reasoning_effort` will be automatically removed from requests before forwarding to this backend.
- This is particularly useful when aliases direct Cursor-optimized prompts (which may include provider-specific parameters) to different backends that don't support those parameters.

#### Combined Example: Aliases, Role Rewrites, and Unsupported Parameters

Here's a complete configuration example illustrating how these features are used together:

```json
{
	"listening_port": 11411,
	"llmrouter_api_key_env": "LLMROUTER_API_KEY",
	"aliases": {
		"o1": "groq/deepseek-r1-distill-qwen-32b",
		"o3-mini": "ollama/qwq"
	},
	"backends": [
		{
			"name": "groq",
			"base_url": "https://api.groq.com/openai/v1",
			"prefix": "groq/",
			"require_api_key": true,
			"key_env_var": "GROQ_API_KEY",
			"role_rewrites": {
				"developer": "system"
			},
			"unsupported_params": [
				"reasoning_effort"
			]
		},
		{
			"name": "ollama",
			"base_url": "http://localhost:11434/v1",
			"prefix": "ollama/",
			"role_rewrites": {
				"developer": "system"
			}
		}
	]
}
```

In this configuration:
- Requests to `o1` from Cursor go to Groq's reasoning-oriented model `deepseek-r1-distill-llama-70b-specdec`.
- Requests to `o3-mini` go to the local Ollama model `qwq`.
- Both backends use role rewriting to map Cursor's custom `developer` role to the standard `system` role.
- The Groq backend drops the `reasoning_effort` parameter, which is not supported by Groq's API.

#### Additional Uses

These features are not limited to reasoning models. They can be applied broadly to facilitate compatibility and optimal prompting strategies across various model types and backend configurations.

### API Keys

Provide the necessary API keys via environment variables:
```sh
OPENAI_API_KEY=<YOUR_OPENAI_KEY> GROQ_API_KEY=<YOUR_GROQ_KEY> ./llm-router-darwin-arm64
```

If you wish to specify your own LLM-router API key instead of using a generated one:
```sh
LLMROUTER_API_KEY=your_custom_key GROQ_API_KEY=<YOUR_GROQ_KEY> ./llm-router-darwin-arm64
```

Alternatively, you can use the command-line flag:
```sh
./llm-router-darwin-arm64 --llmrouter-api-key=your_custom_key
```

#### Using .env Files

You can also store your API keys and other configuration in a `.env` file in the same directory as LLM-router. This is recommended to avoid exposing sensitive keys in your shell history or environment.

1. Create a `.env` file by copying the provided example:
```sh
cp .env.example .env
```

2. Edit the `.env` file with your API keys:
```sh
# LLM-Router configuration
LLMROUTER_API_KEY=your_llmrouter_api_key_here
OPENAI_API_KEY=your_openai_api_key_here
GROQ_API_KEY=your_groq_api_key_here
```

LLM-router will automatically load variables from this file at startup. Environment variables that are already set will take precedence over those in the `.env` file, following standard precedence rules.

## Details

Routes `chat/completions` API requests to any OpenAI-compatible LLM backend based on the model's prefix. Streaming is supported.

LLM-router can be set up to use individual API keys for each backend, or no key for services like local Ollama.

Requests to LLM-router are secured with your `LLMROUTER_API_KEY` which you set in Cursor's OpenAI API Key field. This key is used to authenticate requests to your LLM-router, while the backend-specific API keys (like GROQ_API_KEY) are used by LLM-router to authenticate with the respective API providers.

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
