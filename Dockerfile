ARG IRONLIB_IMAGE=ghcr.io/metal-toolbox/ironlib:v0.2.5
FROM $IRONLIB_IMAGE

COPY vogelkop /usr/sbin/vogelkop
RUN chmod 755 /usr/sbin/vogelkop

ENTRYPOINT ["/usr/sbin/vogelkop"]
CMD ["--help"]
