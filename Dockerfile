FROM scratch

LABEL maintainer "tone@t0ne.net"

COPY bin/sakuin /app/
COPY assets/ /app/assets/

EXPOSE 3000
VOLUME ["/data"]

WORKDIR /app

ENTRYPOINT ["/app/sakuin"]
CMD ["-dir","/data"]
