ARG IRONLIB_REPO=ghcr.io/metal-toolbox
ARG IRONLIB_TAG=sha256:a2237782191cd00fed7e8c2b4e81fe58b52f2a90446621d3459a0f514419aab9
FROM $IRONLIB_REPO/ironlib@$IRONLIB_TAG

COPY vogelkop /usr/sbin/vogelkop
RUN chmod 755 /usr/sbin/vogelkop

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
