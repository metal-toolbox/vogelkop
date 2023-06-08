ARG IRONLIB_REPO=ghcr.io/metal-toolbox
ARG IRONLIB_TAG=sha-046e1c7
FROM $IRONLIB_REPO/ironlib@$IRONLIB_TAG as intermediate

COPY --chmod=755 vogelkop /usr/sbin/vogelkop

FROM scratch
COPY --from=intermediate / /

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
