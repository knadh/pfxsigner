FROM ubuntu:latest AS deploy
WORKDIR /app
COPY pfxsigner .
COPY props.json.sample .
ENTRYPOINT [ "./pfxsigner" ]
