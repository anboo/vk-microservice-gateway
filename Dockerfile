FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go get github.com/gorilla/mux && go get github.com/vorkytaka/easyvk-go/easyvk
RUN go build -o main .
CMD ["/app/main"]
EXPOSE 8000