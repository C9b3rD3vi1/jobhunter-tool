FROM golang:1.21-alpine

WORKDIR /app

# Install Python and required packages for scraping
RUN apk add --no-cache python3 py3-pip git
RUN pip3 install beautifulsoup4 requests

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /jobhunter-ai

EXPOSE 3000

CMD ["/jobhunter-ai"]