FROM golang:1.25

COPY go.sum go.mod ./
RUN go mod download

COPY . .
RUN go build -o /bin/deploy ./cmd/deploy 

EXPOSE 8080

CMD ["/bin/deploy"]
