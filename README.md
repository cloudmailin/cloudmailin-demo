<a href="https://www.cloudmailin.com">
  <img src="https://assets.cloudmailin.com/assets/favicon.png" alt="CloudMailin Logo" height="60" width="60" align="right" title="CloudMailin">
</a>

# CloudMailin Demo Application

A demonstration application for CloudMailin webhook processing and email responses, built with Go.

## Overview

This application was built as a test harness for the
[CloudMailin](https://www.cloudmailin.com) email-to-webhook integration, mainly
to test new hosting environments.

It provides:

* Real-time email webhook processing
* Live updates via WebSocket
* Automatic email response generation
* Simple web interface with Tailwind CSS
* Docker for development

> The application is built to use at least version 1.22+ of Go.

## Getting Started

1. Clone the repository or run the docker container within the hosting environment.
2. Make sure you've got a CloudMailin account to test with.
3. Ensure the environment variables are set:

#### Environment Variables

The application requires the following environment variables:

| Variable                        | Description                           |
|---------------------------------|---------------------------------------|
| `PORT`                          | The port to run the web server on     |
| `CLOUDMAILIN_FORWARD_ADDRESS`   | Your CloudMailin email address        |
| `CLOUDMAILIN_SMTP_URL`          | SMTP URL for sending replies          |
| `FROM_EMAIL`                    | Email address to send replies from    |

## How It Works

1. The application starts a web server that displays your CloudMailin forward address
2. When an email is received at your CloudMailin address, it triggers a webhook to `/emails`
3. The application processes the webhook payload and broadcasts it via WebSocket
4. The web interface updates in real-time to show the new email
5. An automatic reply is sent using the CloudMailin SMTP service

## Development

The application runs in a devcontainer for vs-code / cursor etc. just open
the folder and > Reopen in Container.

The application is built to use at least version 1.22+ of Go.

## Docker

If you just want to build and run the docker container locally:

```bash
# Build the image
docker build -t cloudmailin-example .

# Run the container with env variables from .env file
cp .env.example .env
docker run -d -p 8080:8080 --env-file .env cloudmailin-example
```

OR

```bash
docker compose up --build
```
