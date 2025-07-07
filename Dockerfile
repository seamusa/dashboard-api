FROM golang:alpine as BUILD
ADD app /app
WORKDIR /app
EXPOSE 5000
RUN go build -o app .

FROM alpine
RUN apk add ca-certificates
RUN mkdir -p /app
COPY --from=BUILD /app/app /app
WORKDIR /app
EXPOSE 5000
CMD ["./app"]