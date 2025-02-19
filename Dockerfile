# Build container:
# docker build -t server

# Run container:
# docker run -p 8080:8080 server

# Build Stage
#FROM golang:1.23.1 AS buildstage
FROM golang:1.23.5 AS buildstage

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

# Install any needed dependencies...
RUN go get github.com/gorilla/mux
RUN go get github.com/gorilla/sessions
RUN go get github.com/mattn/go-sqlite3

#COPY . .
COPY . /build
# Build for Linux with static linking
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o /build/server main.go
#ERROR: Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work

# Final Stage
#FROM alpine:latest
FROM golang:1.23.5

WORKDIR /app

# Copy the statically built binary
COPY --from=buildstage /build/static /app/static
COPY --from=buildstage /build/templates /app/templates
COPY --from=buildstage /build/minitwit.db /app/minitwit.db
COPY --from=buildstage /build/server /app/server

# Make port 8080 available to the host
EXPOSE 8080

# Use the binary as the entrypoint
ENTRYPOINT ["/app/server"]