FROM golang:1.23.4

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o gramx .

# Expose the application's port
EXPOSE 8001

# Command to run the application
CMD ["./gramx"]
