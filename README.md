# ReelTalkBot-Go

<img src="https://raw.githubusercontent.com/joelradon/ReelTalkBot-Go/refs/heads/main/image/logo.png" alt="ReelTalkBot-Go Logo" width="33%">

**ReelTalkBot-Go** is a robust Telegram bot built with Go, leveraging OpenAI's language models for intelligent, context-aware responses to user queries. It integrates with AWS S3 for logging user interactions, ensuring data persistence and easy access for analysis.

---

## 📋 Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Bot](#running-the-bot)
- [Project Structure](#project-structure)
- [Logging to AWS S3](#logging-to-aws-s3)
- [Contributing](#contributing)
- [License](#license)
- [Troubleshooting](#troubleshooting)
- [Contact](#contact)

---

## 🚀 Features

- **Intelligent Responses:** Utilizes OpenAI's GPT models for meaningful, context-aware replies.
- **Telegram Integration:** Responds to messages in both private and group chats, supporting mentions.
- **AWS S3 Logging:** Logs all user interactions, including prompts, response times, rate limits, and usage frequency in CSV format.
- **Rate Limiting:** Limits user queries to 10 per 10 minutes, with time remaining until limit reset.
- **Caching and Rate Tracking:** Optimizes performance and prevents redundant API calls, tracking usage history.
- **Secure Configuration:** Manages sensitive data through environment variables and AWS Secrets Manager (optional).

---

## 🔧 Prerequisites

Before beginning, ensure you have:

- **Go** (v1.20 or later): [Download Go](https://golang.org/dl/)
- **AWS Account**: For AWS S3 integration.
- **Telegram Account**: For managing your Telegram bot.
- **OpenAI API Key**: To access OpenAI's language models. [Get an API Key](https://platform.openai.com/account/api-keys)
- **Git**: To clone the repository. [Download Git](https://git-scm.com/downloads)

---

## 🛠️ Installation

### 1. Clone the Repository

```bash
git clone https://github.com/joelr/ReelTalkBot-Go.git
cd ReelTalkBot-Go

```

### 2. Install Dependencies

Ensure you have Go installed (version 1.20 or later). Then, fetch the required dependencies:

```bash
go mod tidy
```

*Note: If you encounter issues with dependencies, refer to the [Troubleshooting](#troubleshooting) section.*

---

## ⚙️ Configuration

The bot requires several environment variables to function correctly. You can manage these using a `.env` file.

### 1. Create a `.env` File

In the root directory of the project, create a `.env` file:

```bash
touch .env
```

### 2. Define Environment Variables

Open the `.env` file in your preferred text editor and add the following variables:

```env
# Telegram Bot Token
TELEGRAM_TOKEN=your_telegram_bot_token

# OpenAI API Key
OPENAI_KEY=your_openai_api_key

# OpenAI Endpoint (optional, defaults to OpenAI's API)
OPENAI_ENDPOINT=https://api.openai.com

# Telegram Bot Username (without @)
BOT_USERNAME=YourBotUsername

# AWS Configuration
AWS_REGION=your_aws_region
AWS_ENDPOINT_URL_S3=https://s3.your-region.amazonaws.com # Modify if using a custom endpoint

# AWS S3 Bucket Name
BUCKET_NAME=your_s3_bucket_name
```

**Ensure you replace the placeholder values with your actual credentials and configurations.**

### 3. Secure Your `.env` File

To prevent sensitive information from being committed to version control, add `.env` to your `.gitignore` file:

```gitignore
# .gitignore

# Environment Variables
.env
```

---

## 🏃‍♂️ Running the Bot

Once you've installed the dependencies and configured the environment variables, you can run the bot using the following command:

```bash
go run ./cmd/main.go
```

*Alternatively, you can build the project and run the executable:*

```bash
go build -o ReelTalkBot-Go ./cmd/main.go
./ReelTalkBot-Go
```

**Note:** Ensure that your AWS credentials are properly configured in your environment or via AWS configuration files to allow the bot to access the S3 bucket.

---

## 📁 Project Structure

```
ReelTalkBot-Go/
├── cmd/
│   └── main.go                 # Entry point of the application
├── internal/
│   ├── app.go                  # Application setup and main logic
│   ├── api_requests.go         # OpenAI API interaction
│   ├── cache.go                # In-memory caching utilities
│   ├── telegram_handler.go     # Telegram message handling
│   ├── s3_client.go            # AWS S3 client setup and logging
│   ├── secrets_manager.go      # AWS Secrets Manager integration (if applicable)
│   ├── types.go                # Type definitions for API interactions
│   ├── usage_cache.go          # User rate-limiting cache and tracking
│   └── utils.go                # Utility functions
├── go.mod                      # Go module file
├── go.sum                      # Go checksum file
├── .env                        # Environment variables (not committed)
├── .gitignore                  # Git ignore rules
└── README.md                   # Project documentation

```

---

## 📊 Logging to AWS S3

ReelTalkBot-Go logs all user interactions to an AWS S3 bucket in CSV format. Each log entry includes:

- `userID`: Telegram user ID
- `username`: Telegram username
- `prompt`: User's message
- `responseTimeMS`: Time taken to generate a response in milliseconds
- `queryCount`: Number of queries in the last 10 minutes
- `isRateLimited`: Indicates if the user is currently rate-limited

### 1. Set Up AWS S3 Bucket

1. **Create an S3 Bucket:**
   - Log in to your AWS Management Console.
   - Navigate to S3 and create a new bucket (e.g., `reeltalkbot-logs`).
   - Configure permissions and access policies as needed.

2. **Configure AWS Credentials:**
   - Ensure that your AWS credentials (`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`) are set in your environment or via AWS configuration files.
   - Alternatively, use IAM roles if deploying on AWS infrastructure.

### 2. Verify Logging

After running the bot, navigate to your S3 bucket and check the `logs/telegram_logs.csv` file to ensure that logs are being recorded correctly.

---

## 🤝 Contributing

Contributions are welcome! To contribute to **ReelTalkBot-Go**, follow these steps:

1. **Fork the Repository**
2. **Create a New Branch**
   ```bash
   git checkout -b feature/YourFeature
   ```
3. **Make Your Changes**
4. **Commit Your Changes**
   ```bash
   git commit -m "Add Your Feature"
   ```
5. **Push to the Branch**
   ```bash
   git push origin feature/YourFeature
   ```
6. **Open a Pull Request**

Please ensure your code adheres to the project's coding standards and passes all tests.

---

## 📝 License

This project is licensed under the [MIT License](LICENSE).

---

## 🛠️ Troubleshooting

### **1. Go Module Errors**

If you encounter issues related to `go.mod`, such as invalid module paths or versions:

- **Ensure Correct Module Paths:**  
  Verify that all module paths in `go.mod` are accurate and refer to existing modules.

- **Clear Module Cache:**
  ```bash
  go clean -modcache
  ```

- **Set Go Proxy:**
  ```bash
  go env -w GOPROXY=https://proxy.golang.org,direct
  ```

- **Fetch Dependencies Again:**
  ```bash
  go get -u ./...
  go mod tidy
  ```

### **2. Duplicate Method Declarations**

If you receive errors about duplicate method declarations:

- **Review Code for Duplicates:**  
  Open the affected `.go` files and ensure that each method is uniquely defined.

- **Remove or Rename Duplicates:**  
  If a method is defined multiple times, remove the redundant ones or rename them to reflect different functionalities.

### **3. AWS S3 Logging Issues**

If logs are not appearing in your S3 bucket:

- **Verify AWS Credentials:**  
  Ensure that the bot has the necessary permissions to write to the S3 bucket.

- **Check Bucket Configuration:**  
  Confirm that the bucket name and region are correctly specified in the `.env` file.

- **Review Application Logs:**  
  Check the bot's logs for any errors related to AWS S3 operations.

### **4. Telegram Bot Not Responding**

If the bot isn't responding to messages:

- **Verify Webhook Setup:**  
  Ensure that your Telegram bot's webhook is correctly set to your server's URL.

  ```bash
  https://api.telegram.org/bot<TELEGRAM_TOKEN>/setWebhook?url=<YOUR_PUBLIC_URL>/webhook
  ```

- **Check Server Accessibility:**  
  Ensure that your server is publicly accessible and that no firewall rules are blocking Telegram's requests.

- **Review Application Logs:**  
  Look for any errors or warnings in the bot's logs that might indicate issues with message handling.

---

## 📫 Contact

For any questions or support, feel free to reach out:

-
- **GitHub:** [@joelradon](https://github.com/joelradon)

---

**Happy Coding!** 🚀

---