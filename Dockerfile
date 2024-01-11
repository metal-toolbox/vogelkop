ARG IRONLIB_REPO=ghcr.io/metal-toolbox
ARG IRONLIB_TAG=v0.2.17@sha256:5b7cdd0efef9bd3242d94c7118bc0c94fa988f31de5fd11a81887da0ccb87235
FROM $IRONLIB_REPO/ironlib:$IRONLIB_TAG as intermediate

COPY --chmod=755 vogelkop /usr/sbin/vogelkop

FROM scratch
COPY --from=intermediate / /

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
