ARG IRONLIB_REPO=ghcr.io/metal-toolbox
ARG IRONLIB_TAG=v0.3.0@sha256:f94599c35126ef6114e4b38ec3f53febb26b9198d7304e0abd2d000e2abcb0da
FROM $IRONLIB_REPO/ironlib:$IRONLIB_TAG as intermediate

COPY --chmod=755 vogelkop /usr/sbin/vogelkop

FROM scratch
COPY --from=intermediate / /

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
