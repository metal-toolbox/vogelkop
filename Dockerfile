ARG IRONLIB_REPO=ghcr.io/metal-toolbox
ARG IRONLIB_TAG=v0.2.12@sha256:a4e981d5a3c217cc89a14a02b9663bb47c4746aa0d754bccdcc2b4edd316575f
FROM $IRONLIB_REPO/ironlib:$IRONLIB_TAG as intermediate

COPY --chmod=755 vogelkop /usr/sbin/vogelkop

FROM scratch
COPY --from=intermediate / /

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
