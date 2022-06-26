FROM scratch

LABEL maintainer "tone@t0ne.net"

COPY sakuin /app/

EXPOSE 3000
VOLUME ["/data"]

WORKDIR /app

ENV SAKUIN_DATA_DIR="${SAKUIN_DATA_DIR:-/data}"

ENTRYPOINT ["/app/sakuin", "serve"]
