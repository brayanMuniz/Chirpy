# Use a minimal base image 
FROM alpine:latest

# Install necessary tools
RUN apk add --no-cache bash postgresql-client

# Set the working directory
WORKDIR /app

COPY ./sql/init.sql /docker-entrypoint-initdb.d/

# Copy the precompiled binary into the container
COPY ./Chirpy /app/Chirpy

# Create a directory for static files and copy them there
RUN mkdir -p /app/static
COPY ./index.html /app/static/index.html

EXPOSE 8080

# Set the default command 
CMD ["./Chirpy"]

