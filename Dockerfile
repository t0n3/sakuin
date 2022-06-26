FROM scratch

LABEL maintainer "tone@t0ne.net"

WORKDIR /app

COPY sakuin .

EXPOSE 3000
VOLUME ["/data"]

ENV SAKUIN_DATA_DIR="${SAKUIN_DATA_DIR:-/data}"

ENTRYPOINT ["/app/sakuin", "serve"]
