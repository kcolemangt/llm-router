# Changelog

All notable changes to this project will be documented here.

## [0.2.1] - 2025-03-25

### Fixed
- Enhanced streaming response handling and proxy error management
- Improved Content-Length handling and response logging

## [0.2.0] - 2025-03-17

### Added
- Support for sending optimized reasoning prompts to local and alternative model providers via model aliasing and role rewriting:
  - Model aliases allow remapping client-recognized reasoning models to your chosen backend models.
  - Role rewrites enable client-specific roles to be translated into backend-compatible roles, ensuring prompt compatibility across providers.
- Support for loading environment variables from `.env` files using godotenv:
  - Automatically loads variables from `.env` file in the current directory.
  - Follows standard precedence rules where manually set environment variables take priority over `.env` values.
  - Simplifies configuration without exposing sensitive API keys in your shell history.

## [0.1.0] - 2025-03-17

### Added
- Simplified initial setup and Cursor configuration process
- Better detection of streaming requests in `/v1/chat/completions`
- New environment variable approach for setting the LLM Router API key
- Automatic API key generation when no key is provided
- Updated readme instructions on configuring Cursor and environment variables
- Improved log details for both incoming and outgoing requests

### Changed
- Removed need for `/v1` suffix in Cursor's OpenAI Base URL setting (now included in provider configs)
- Renamed `GlobalAPIKey` to `LLMRouterAPIKey` in the config for clarity
- Switched from `GlobalAPIKeyEnv` to `LLMRouterAPIKeyEnv`
- Revised config loading and command-line parsing
- Updated default `go.mod` to use Go 1.24

### Removed
- Old references to `GlobalAPIKey` environment variable

---

This is the first official changelog entry. Previous changes up to `v0.0.1` were not tracked in this format.
