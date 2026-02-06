FROM alpine:3.19

# Install bash + ssh client
RUN apk add --no-cache bash openssh

WORKDIR /app

# Copy scripts + host list
COPY run-on-remotes.sh hosts.txt reset-ubuntu.sh ./

# Make scripts executable
RUN chmod +x run-on-remotes.sh reset-ubuntu.sh

# Default command
ENTRYPOINT ["bash", "./run-on-remotes.sh", "./hosts.txt", "./reset-ubuntu.sh"]
