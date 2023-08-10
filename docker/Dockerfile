# Use a lightweight base image for Go
FROM golang:1.17-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download and cache the Go dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o /bin/app ./cmd

# Use a minimal base image for the final application
FROM alpine:3.14

# Set the working directory inside the container
WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=0 /bin/app /bin/app

ENTRYPOINT [ "/bin/app" ]
