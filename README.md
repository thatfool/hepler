# hepler

Hepler is a small LLM integration for your shell. Press a key combination (default: ctrl-g) to have it double-check your command line, fix typos, or implement requests you write in a comment.

It uses LLMs via the OpenAI API, and supports many online providers, as well as local options such as Ollama or LM Studio.

## Installation

### Prerequisites

- go 1.26

### Build & Install

Run `make install` to build for the host system and install to the standard go binaries location. Make sure it's on your PATH.

### Building Releases

Run `make release` to build hepler packages for macOS and Linux, on aarch64 and amd64. The packages are stored in `dist/`.
Extract the one matching your platform and copy the ececutable to a reasonable location that's on your PATH.

### Setting up Shell Integration

Hepler will tell you how to integrate it in your shell. Run `hepler init <name of your shell>` (e.g., `hepler init bash`) for a snippet you can put in your Shell's configuration.

## LLM configuration

Hepler works with any OpenAI-compatible LLM provider. Since tasks are not very complex, local models are a good option.

LLM access is configured through environment variables:

- `HEPLER_OPENAI_API_BASE` — the base URL of your OpenAI-compatible endpoint (e.g. `http://localhost:11434/v1` for Ollama, or `http://localhost:1234/v1` for LM Studio).
- `HEPLER_OPENAI_API_KEY` — your API key for that endpoint.
- `HEPLER_MODEL_NAME` — the model to use (e.g. `google/gemma-4-12b-qat`).

## License

Licensed under the terms of the MIT license as written in the file LICENSE.
