ARG IRONLIB_REPO=ghcr.io/metal-toolbox
FROM $IRONLIB_REPO/ironlib@sha256:c06b454b0264a6837cefaf5ba933d0d50c249aa8cbfec6c2ce2e76decb65a02f

COPY vogelkop /usr/sbin/vogelkop
RUN chmod 755 /usr/sbin/vogelkop

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
