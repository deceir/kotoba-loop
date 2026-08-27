FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /kotoba-api ./cmd/api
RUN CGO_ENABLED=0 go build -o /kotoba-import ./cmd/import-jlpt
FROM alpine:3.22
COPY --from=build /kotoba-api /kotoba-api
COPY --from=build /kotoba-import /kotoba-import
EXPOSE 8080
CMD ["/kotoba-api"]
