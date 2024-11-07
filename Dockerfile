FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy the Go files to the working directory
COPY . .

# Build the Go app
RUN go build -o main .

# Use a minimal base image for the final container
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the compiled Go binary and hostsfile.txt from the builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/hostsfile-testcase1.txt .
COPY --from=builder /app/hostsfile-testcase2.txt .

# Run the Go binary when the container starts
ENTRYPOINT [ "./main" ]