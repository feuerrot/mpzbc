FROM golang:1.15                                         
COPY . /src
RUN cd /src && go build -v .

FROM debian:buster-slim
COPY --from=0 /src/mpzbc /mpzbc

CMD ["/mpzbc"]