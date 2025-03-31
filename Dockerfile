ARG IRONLIB_REPO=ghcr.io/metal-toolbox
ARG IRONLIB_TAG=v0.5.0@sha256:3cdf3401bb3661c4c2abc01dbf131f693a2aea01e9b302335c02a0b29b488fa2
FROM $IRONLIB_REPO/ironlib:$IRONLIB_TAG as intermediate

COPY --chmod=755 vogelkop /usr/sbin/vogelkop

FROM scratch
COPY --from=intermediate / /

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
