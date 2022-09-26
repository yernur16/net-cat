FROM golang:1.17-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o servak main.go
FROM alpine
WORKDIR /app
EXPOSE 8989
COPY --from=build /app .
CMD /app/servak