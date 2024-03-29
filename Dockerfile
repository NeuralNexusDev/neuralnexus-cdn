FROM alpine:3.14

COPY ./simple-cdn .

CMD ["./simple-cdn"]
