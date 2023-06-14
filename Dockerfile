ARG IRONLIB_REPO=ghcr.io/metal-toolbox
ARG IRONLIB_TAG=v0.2.11@sha256:8b3c5a78a2b323860b0525723aad7451a7e026a5e552b3ce945522762fe0870d
FROM $IRONLIB_REPO/ironlib:$IRONLIB_TAG as intermediate

COPY --chmod=755 vogelkop /usr/sbin/vogelkop

FROM scratch
COPY --from=intermediate / /

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
