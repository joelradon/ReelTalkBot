# ReelTalkBot-Go

ReelTalkBot-Go is a Telegram chatbot focused on providing fishing information to users. This bot integrates with Azure OpenAI and a Custom Question Answering (CQA) service to respond to fishing-related queries. It also utilizes caching to limit API calls and manage rate limits, especially for high-usage scenarios.

## Table of Contents
- [Folder Structure](#folder-structure)
- [Project Setup](#project-setup)
- [Environment Variables](#environment-variables)
- [Application Files](#application-files)
  - [cmd/main.go](#cmdmaingo)
  - [internal/app.go](#internalappgo)
  - [internal/api_requests.go](#internalapi_requestsg)
  - [internal/cache.go](#internalcachego)
  - [internal/secrets_manager.go](#internalsecrets_managergo)
  - [internal/telegram_handler.go](#internaltelegram_handlergo)
- [How It Works](#how-it-works)
- [API Endpoints](#api-endpoints)

---

## Folder Structure

The application has the following folder and file structure:



## Folder Structure
```
ReelTalkBot-Go/- 
cmd/
  - main.go            # Main entry point for starting the server- 
  
  internal/
  - app.go             # Main application setup and configuration
  - api_requests.go    # Handles requests to OpenAI services
  - cache.go           # Implements caching to reduce API calls
  - secrets_manager.go # Manages environment variable-based secrets
  - telegram_handler.go # Processes incoming Telegram messages- Dockerfile           # Docker setup for containerized deployment- go.mod               # Go module dependencies- go.
  
  sum               # Go module checksums
```
---
## Project Setup
 1. **Clone the repository:**
   ```bash
   git clone https://github.com/username/ReelTalkBot-Go.git
   cd ReelTalkBot-Go
   ```
 2. **Install dependencies**: 
 
 Ensure you have Go installed. Then, download dependencies:
   ```bash
   go mod tidy
   ```

 3. **Set up environment variables**:
 
  Create a `.env` file to configure required environment variables (see Environment
 Variables).

 4. **Run the application**:

   ```bash
   go run cmd/main.go
   ```
 ## Environment Variables

 The application requires the following environment variables:- `TELEGRAM_TOKEN`: Telegram bot token.
- `CQA_KEY`: API key for Custom Question Answering.- `CQA_ENDPOINT`: Endpoint for Custom Question Answering.- `OPENAI_KEY`: API key for OpenAI.- `OPENAI_ENDPOINT`: Endpoint for OpenAI.

 ## Application Files

 ### cmd/main.go
 The main entry point of the application. It initializes the application by calling `NewApp()` and starts the server, exposing
 an API endpoint at `/api/FishingBotFunction`.
 **Key Sections**:- 
 **HTTP Server**: Configures and starts an HTTP server on port 8080, using the handler function `app.Handler`.
 
 ### internal/app.go
 This file defines the main `App` structure, which holds the configuration and dependencies required by the application.
 **Key Sections**:- 
 **NewApp Function**: Initializes and returns an `App` instance by loading environment variables and configuring the
 cache.- 
 **App Struct**: Holds configurations and API keys required by the bot, such as `TelegramToken`, `CQAKey`,
 `CQAEndpoint`, `OpenAIKey`, `OpenAIEndpoint`, and an in-memory `Cache` for reducing API calls.
 
 ### internal/api_requests.go
 Handles requests to OpenAI and the Custom Question Answering service.
**Key Sections**:- 
**QueryOpenAIWithCache Function**:
 Manages requests to OpenAI, using caching to prevent excessive API calls.
 Caches responses for 30 minutes.- 
 
 **QueryCQA Function**: 
 Sends a request to the Custom Question Answering API. The function is called if a specific
 question matches CQA; otherwise, the bot queries OpenAI.

 ### internal/cache.go
 Implements caching to store responses temporarily and reduce API requests. This file defines the `Cache` struct and
 associated methods for setting and getting cached responses.

 **Key Sections**:- 
 **Cache Struct**: Stores cached entries, each with an expiration time.- 
 **Get and Set Methods**: Manages storing responses and retrieving them if they are still valid.

 ### internal/secrets_manager.go
 Manages secrets loaded from environment variables, ensuring they are accessible throughout the application.

 **Key Sections**:
 - **Environment Variables**: Retrieves environment variables for Telegram, OpenAI, and CQA API keys and endpoints.- 
 **NewApp Configuration**: Loads variables during app initialization in `app.go`.

 ### internal/telegram_handler.go
 Processes incoming messages from Telegram, first attempting to answer using cached or Custom Question Answering
 responses, then querying OpenAI if needed.

 **Key Sections**:
- **Handler Function**: The main HTTP handler for incoming messages from Telegram.- **HandleTelegramMessage Function**: Processes each message by first checking the cache, then querying CQA, and
 finally querying OpenAI if necessary.
 ## How It Works
 1. **Message Reception**: A message sent by a user in Telegram is received by `HandleTelegramMessage` in
 `telegram_handler.go`.
 2. **Cache Lookup**: The bot checks the cache for a matching response to avoid unnecessary API calls.
 3. **Custom Question Answering (CQA)**: If there is no cached response, it checks if the question is a match in the
 Custom Question Answering API.
 4. **OpenAI Query**: If neither cache nor CQA has a response, it queries OpenAI and caches the response for future
 queries.
 5. **Rate Limiting and 429 Management**: Uses caching and backoff retry to handle rate limits, with a wait time to avoid
 consecutive 429 errors.
 ## API Endpoints- `/api/FishingBotFunction`
  This endpoint receives messages from Telegram, processes them using `HandleTelegramMessage` in
 `telegram_handler.go`, and sends a response back to the user.
 ## Docker Deployment
 To build and run the application in a Docker container:
### Build Docker Image:
   ```bash
   docker build -t reeltalkbot-go .
   ```
 ### Run Docker Container:
   ```bash
   docker run -d -p 8080:8080 --env-file .env reeltalkbot-go
   ```
 ## Troubleshooting- **429 Too Many Requests**: Adjust the cache duration or reduce API call frequency.- **Telegram Handler Undefined**: Ensure you use `app.Handler` in `main.go`.- **Environment Variable Issues**: Confirm all environment variables are set correctly.
 This documentation covers the details and setup of each application component. If there are further questions or
 customizations needed, please refer to the comments within the source code or contact the maintainers.