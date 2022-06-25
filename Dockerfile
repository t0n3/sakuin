FROM scratch

LABEL maintainer "tone@t0ne.net"

COPY build/sakuin /app/

EXPOSE 3000
VOLUME ["/data"]

WORKDIR /app

ENTRYPOINT ["/app/sakuin", "serve"]
