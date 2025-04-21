# SSHTalk TUI with SSH Support

A terminal-based chat application with OpenAI integration that also supports SSH access.

## Features

- Chat with OpenAI models directly from your terminal
- Access the chat application via SSH
- Stylish TUI interface using Bubble Tea

## Setup

1. Clone the repository
2. Set the required environment variables:
   ```
   export OPENAI_API_KEY=your_openai_api_key
   export OPENAI_MODEL=gpt-4-turbo  # or another OpenAI model
   export OPENAI_BASE_URL=https://api.openai.com/v1  # Optional, defaults to OpenAI's API
   ```
3. Generate SSH keys for the server:
   ```
   ./generate_ssh_key.sh
   ```

## Running the Application

### Direct Terminal Mode

To run the application directly in your terminal:

```
go run .
```

### SSH Server Mode

To run the application as an SSH server:

```
go run . -ssh
```

By default, the SSH server runs on port 2222. You can change this by setting the `SSH_ADDR` environment variable:

```
export SSH_ADDR=:2223
go run . -ssh
```

### Connecting via SSH

Connect to the running SSH server using:

```
ssh -p 2222 localhost
```

## Building the Application

To build the application:

```
go build
```

Then run it:

```
# Direct terminal mode
./sshtalk

# SSH server mode
./sshtalk -ssh
```

## Docker

### Running with Docker

You can run the application using Docker:

```bash
# Build the image
docker build -t sshtalk .

# Run in direct terminal mode (interactive)
docker run -it --rm -e OPENAI_API_KEY=your_api_key -e OPENAI_MODEL=gpt-4-turbo sshtalk

# Run in SSH server mode
docker run -d -p 2222:22 -e OPENAI_API_KEY=your_api_key -e OPENAI_MODEL=gpt-4-turbo sshtalk
```

### Using Docker Compose

To run with Docker Compose:

```bash
# Create a .env file with your API key and model
echo "OPENAI_API_KEY=your_api_key" > .env
echo "OPENAI_MODEL=gpt-4-turbo" >> .env

# Start the services
docker-compose up -d
```

## Continuous Integration/Deployment

This project uses GitHub Actions for continuous integration and delivery. On every push to the main branch or when a new tag is created, the workflow automatically:

1. Builds a Docker image
2. Pushes the image to GitHub Container Registry (ghcr.io)

To use the container images from the registry:

```bash
docker pull ghcr.io/USERNAME/sshtalk:latest
```

Replace `USERNAME` with your GitHub username.

When creating a new release, tag your commit with a version number (e.g., `v1.0.0`) and the workflow will automatically build and publish a versioned Docker image. 