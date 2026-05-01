FROM golang:1.22-alpine

# Install git and ssh-client
RUN apk add --no-cache git openssh-client

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Build the application
RUN go build -o mirror-bot

# Create .ssh directory and set permissions
RUN mkdir -p /root/.ssh && chmod 700 /root/.ssh

# Add known hosts for github.com, gitlab.com, and codeberg.org
RUN ssh-keyscan github.com gitlab.com codeberg.org >> /root/.ssh/known_hosts

# Run the application
CMD ["./mirror-bot"]