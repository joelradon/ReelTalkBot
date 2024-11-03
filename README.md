<img src="https://raw.githubusercontent.com/joelradon/ReelTalkBot/refs/heads/main/image/logo.png" alt="ReelTalkBot Logo" width="33%">

# ReelTalkBot-Go

**ReelTalkBot-Go** is a powerful Telegram bot developed in Go, leveraging OpenAI's language models to provide intelligent, context-aware responses to user queries. The bot integrates with AWS S3 for logging user interactions, ensuring data persistence and easy access for analysis.

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
- **AWS S3 Logging:** Logs all user interactions in CSV format, including prompts, response times, rate limits, and usage frequency.
- **Rate Limiting:** Limits user queries to 10 per 10 minutes, with remaining time until limit reset.
- **Caching and Rate Tracking:** Optimizes performance and prevents redundant API calls by tracking usage history.
- **Secure Configuration:** Manages sensitive data through environment variables and AWS Secrets Manager (optional).

---

## 🛠 Prerequisites

Before beginning, ensure you have:

- **Go** (v1.20 or later): [Download Go](https://golang.org/dl/)
- **AWS Account:** For AWS S3 integration.
- **Telegram Account:** For managing your Telegram bot.
- **OpenAI API Key:** To access OpenAI's language models. [Get an API Key](https://platform.openai.com/account/api-keys)
- **Git:** To clone the repository. [Download Git](https://git-scm.com/downloads)

---

## ⚙️ Installation

### 1. Clone the Repository

```bash
git clone https://github.com/joelradon/ReelTalkBot-Go.git
cd ReelTalkBot-Go
2. Install Dependencies
Ensure you have Go installed (version 1.20 or later). Then, fetch the required dependencies:

bash
Copy code
go mod tidy
Note: If you encounter issues with dependencies, refer to the Troubleshooting section.

📝 Configuration
The bot requires several environment variables to function correctly. You can manage these using a .env file.

1. Create a .env File
In the root directory of the project, create a .env file:

bash
Copy code
touch .env
2. Define Environment Variables
Open the .env file in your preferred text editor and add the following variables:

env
Copy code
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

# NO_LIMIT_USERS (Comma-separated user IDs without spaces for no rate limit)
NO_LIMIT_USERS=12345678,87654321

# KNOWLEDGE_BASE (Set to ON to enable Knowledge Base queries)
KNOWLEDGE_BASE=OFF

# KNOWLEDGE_BASE_TRAIN_ENDPOINT (Optional, for training)
KNOWLEDGE_BASE_TRAIN_ENDPOINT=https://your-knowledgebase-app.fly.dev/api/knowledge

# API_KEY for Knowledge Base (if applicable)
API_KEY=your_knowledge_base_api_key
Ensure you replace the placeholder values with your actual credentials and configurations.

3. Secure Your .env File
To prevent sensitive information from being committed to version control, add .env to your .gitignore file:

gitignore
Copy code
# .gitignore

# Environment Variables
.env
🏃 Running the Bot
Once you've installed the dependencies and configured the environment variables, you can run the bot using the following command:

bash
Copy code
go run ./cmd/main.go
Alternatively, you can build the project and run the executable:

bash
Copy code
go build -o ReelTalkBot ./cmd/main.go
./ReelTalkBot
Note: Ensure that your AWS credentials are properly configured in your environment or via AWS configuration files to allow the bot to access the S3 bucket.

📁 Project Structure
bash
Copy code
ReelTalkBot-Go/
├── cmd/
│   └── main.go                  # Entry point of the application
├── internal/
│   ├── app/
│   │   └── app.go               # Application setup and main logic
│   ├── api/
│   │   └── api_requests.go      # OpenAI API interaction
│   ├── cache/
│   │   └── cache.go             # In-memory caching utilities
│   ├── conversation/
│   │   └── conversation_cache.go# Conversation context management
│   ├── telegram/
│   │   └── telegram_handler.go  # Telegram message handling
│   ├── s3/
│   │   └── s3_client.go         # AWS S3 client setup and logging
│   ├── secrets/
│   │   └── secrets_manager.go   # AWS Secrets Manager integration
│   ├── knowledgebase/
│   │   └── knowledgebase.go     # Knowledge Base client and interactions
│   ├── types/
│   │   └── types.go             # Shared type definitions
│   ├── usage/
│   │   └── usage_cache.go       # User rate-limiting cache and tracking
│   └── utils/
│       └── utils.go             # Utility functions
├── go.mod
├── go.sum
├── .env
├── .gitignore
├── README.md
└── image/
    └── logo.png
📊 Logging to AWS S3
ReelTalkBot logs all user interactions to an AWS S3 bucket in CSV format. Each log entry includes:

userID: Telegram user ID
username: Telegram username
prompt: User's message
responseTimeMS: Time taken to generate a response in milliseconds
queryCount: Number of queries in the last 10 minutes
isRateLimited: Indicates if the user is currently rate-limited
1. Set Up AWS S3 Bucket
Create an S3 Bucket:

Log in to your AWS Management Console.
Navigate to S3 and create a new bucket (e.g., reeltalkbot-logs).
Configure permissions and access policies as needed.
Configure AWS Credentials:

Ensure that your AWS credentials (AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY) are set in your environment or via AWS configuration files.
Alternatively, use IAM roles if deploying on AWS infrastructure.
2. Verify Logging
After running the bot, navigate to your S3 bucket and check the logs/telegram_logs.csv file to ensure that logs are being recorded correctly.

🛠️ Contributing
Contributions are welcome! To contribute to ReelTalkBot-Go, follow these steps:

Fork the Repository
Create a New Branch
bash
Copy code
git checkout -b feature/YourFeature
Make Your Changes
Commit Your Changes
bash
Copy code
git commit -m "Add Your Feature"
Push to the Branch
bash
Copy code
git push origin feature/YourFeature
Open a Pull Request
Please ensure your code adheres to the project's coding standards and passes all tests.

📜 License
This project is licensed under the MIT License.

🛠️ Troubleshooting
1. Go Module Errors
If you encounter issues related to go.mod, such as invalid module paths or versions:

Ensure Correct Module Paths:
Verify that all module paths in go.mod are accurate and refer to existing modules.

Clear Module Cache:

bash
Copy code
go clean -modcache
Set Go Proxy:

bash
Copy code
go env -w GOPROXY=https://proxy.golang.org,direct
Fetch Dependencies Again:

bash
Copy code
go get -u ./...
go mod tidy
2. Duplicate Method Declarations
If you receive errors about duplicate method declarations:

Review Code for Duplicates:
Open the affected .go files and ensure that each method is uniquely defined.

Remove or Rename Duplicates:
If a method is defined multiple times, remove the redundant ones or rename them to reflect different functionalities.

3. AWS S3 Logging Issues
If logs are not appearing in your S3 bucket:

Verify AWS Credentials:
Ensure that the bot has the necessary permissions to write to the S3 bucket.

Check Bucket Configuration:
Confirm that the bucket name and region are correctly specified in the .env file.

Review Application Logs:
Check the bot's logs for any errors related to AWS S3 operations.

4. Telegram Bot Not Responding
If the bot isn't responding to messages:

Verify Webhook Setup:
Ensure that your Telegram bot's webhook is correctly set to your server's URL.

bash
Copy code
https://api.telegram.org/bot<TELEGRAM_TOKEN>/setWebhook?url=<YOUR_PUBLIC_URL>/webhook
Check Server Accessibility:
Ensure that your server is publicly accessible and that no firewall rules are blocking Telegram's requests.

Review Application Logs:
Look for any errors or warnings in the bot's logs that might indicate issues with message handling.

📚 Training the ReelTalkBot with the /learn Command
Overview
The /learn command in the ReelTalkBot application allows authorized users to train the bot with new knowledge entries. This feature enhances the bot's capabilities by adding relevant information that can be retrieved in future interactions. Users can input training data related to bodies of water, fish species, water types, question templates, and answers.

How to Use the /learn Command
Prerequisites
You must be an authorized user to access the training feature. Authorized users are defined in the environment variable NO_LIMIT_USERS within the bot's configuration.
Ensure the knowledge base feature is activated by setting the environment variable KNOWLEDGE_BASE to ON.
Command Format
The /learn command should be formatted as follows:

bash
Copy code
/learn [training data]
training data: This is the information you want to teach the bot. It should include relevant details such as the body of water, fish species, water type, question template, and the corresponding answer.
Example Usage
To train the bot with a new knowledge entry, you might use the command like this:

css
Copy code
/learn I want to learn about the Salmon River. What species are common here? The common species include King Salmon and Coho Salmon.
Command Breakdown
Trigger Command: Start with the /learn command to indicate that you want to train the bot.
Provide Training Data: Follow the command with a clear and structured training sentence that includes:
A reference to the body of water (e.g., "Salmon River").
The question template or the question you want the bot to recognize.
The answer or information related to the question.
Bot Response
After submitting the /learn command with training data, the bot will respond with a confirmation message, indicating that the training data has been received and is being processed. Here’s what you might expect:

kotlin
Copy code
Training data received and is being processed under category: [Category].
If you are not authorized, the bot will respond with:

csharp
Copy code
You are not authorized to train the knowledge base.
If the knowledge base feature is turned off, the response will be:

csharp
Copy code
Knowledge base training is currently disabled.
Important Notes
Rate Limits: Authorized users may still be subject to rate limits defined in the application. For example, the bot limits queries to 10 per 10 minutes.
Message Formatting: Ensure that your training data is clear and concise to avoid confusion during the training process.
Error Handling: If there’s an error in processing the training data, the bot will inform you with an appropriate error message.
Conclusion
Using the /learn command is a powerful way to enhance the ReelTalkBot's knowledge and responsiveness. By adding specific information through training, users can make the bot a more effective assistant for inquiries related to fishing and aquatic life.

📞 Contact
For any questions or support, feel free to reach out:

GitHub: @joelradon
Happy Coding! 🎉
