FROM golang:1.22.2-alpine3.18 AS builder

# Set destination for COPY
WORKDIR /app

# Set environment variables
ENV DATABASE_URI="mongodb+srv://nadeeshaniawwa:qJAGwYojLeM1g7Zv@cluster0.0stniyu.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"

# copy project files
COPY . .
COPY cmd/main.go .

# Build the Go application
RUN  go build -o main .

FROM alpine:3.14

WORKDIR /app


COPY . .
COPY --from=builder /app/main .

ENV DATABASE_URI="mongodb+srv://nadeeshaniawwa:qJAGwYojLeM1g7Zv@cluster0.0stniyu.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"

EXPOSE 3034
CMD ["./main"]